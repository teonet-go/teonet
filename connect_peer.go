// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Connect to peer module

package teonet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"

	"github.com/kirill-scherba/trudp"
)

// ConnectTo connect to any teonet Peer(client or server) by address (client
// sent request to teonet auth server):
// Client call ConnectTo wich send request to teonet auth server and wait
// function connectToAnswerProcess called -> Server call ConnectToProcess send
// infor to Peer and send answer to client (connectToAnswerProcess func called
// on client side when answer received) -> Client connect to Peer and send
// clients teonet address to it, Peer check it in connectToConnected func
func (teo Teonet) ConnectTo(addr string) (err error) {
	// TODO: check local connection exists

	// Local IDs and port
	ips, _ := teo.getIPs()
	port := teo.trudp.Port()

	// Connect data
	conIn := ConnectToData{
		ID:        trudp.RandomString(35),
		Addr:      addr,
		LocalIPs:  ips,
		LocalPort: uint32(port),
	}
	data, _ := conIn.MarshalBinary()

	// Send command to teonet
	teo.Command(CmdConnectTo, data).Send(teo.auth)
	teo.log.Println("send CmdConnectTo", addr, "(send to teo.auth)", ips, port)

	return
}

// ConnectToProcess check connection to teonet peer and send answer to client
// and request to Peer (auth server receive connection data from client)
func (teo Teonet) ConnectToProcess(c *Channel, data []byte) (err error) {

	// Unmarshal data
	var con ConnectToData
	err = con.UnmarshalBinary(data)
	if err != nil {
		teo.log.Println("unmarshal error:", err)
		return
	}
	teo.log.Println("got CmdConnectTo", con.Addr, "from", c, "ID:", con.ID)

	// Get channel data and prepare answer data to Client
	ch, ok := teo.Channel(con.Addr)
	if ok {
		// Server IPs and port
		con.IP = ch.c.UDPAddr.IP.String()
		con.Port = uint32(ch.c.UDPAddr.Port)
	} else {
		con.Err = []byte("address not connected")
	}

	// Prepare data and send request to Peer
	if len(con.Err) == 0 {
		var conPeer ConnectToData
		// Client address, IPs and port
		conPeer.ID = con.ID
		conPeer.Addr = c.a
		conPeer.IP = c.c.UDPAddr.IP.String()
		conPeer.Port = uint32(c.c.UDPAddr.Port)
		data, err = conPeer.MarshalBinary()
		if err != nil {
			return
		}
		// Send request to Peer
		_, err = teo.Command(byte(2), data).SendAnswer(ch)
		if err != nil {
			return
		}
	}

	// Send answer to Client
	data, err = con.MarshalBinary()
	if err != nil {
		return
	}
	_, err = teo.Command(CmdConnectTo, data).SendAnswer(c)

	return
}

// connectToPeer peer got ConnectTo request from teonet auth (peer prepare to
// connection from client and send answer to auth server)
// TODO: send answer to auth server
func (teo Teonet) connectToPeer(data []byte) (err error) {
	// Unmarshal data
	var con ConnectToData
	err = con.UnmarshalBinary(data)
	if err != nil {
		teo.log.Println("connectToPeer unmarshal error:", err)
		return
	}
	teo.log.Println("got CmdConnectToPeer command, data len:", len(data), "ID:", con.ID)
	teo.peerRequests.add(&con)
	return
}

// connectToAnswerProcess check ConnectTo answer from auth server, connect to
// Peer and send clients teonet addres to it (client processed)
func (teo Teonet) connectToAnswerProcess(data []byte) (err error) {

	const cantConnectToPeer = "can't connect to peer, error: "

	// Unmarshal data
	var con ConnectToData
	err = con.UnmarshalBinary(data)
	if err != nil {
		teo.log.Println("connectToAnswerProcess unmarshal error:", err)
		return
	}
	teo.log.Println("got CmdConnectTo answer:", con.Addr, con.IP, con.Port,
		string(con.Err))

	// Check server error
	if len(con.Err) != 0 {
		err = errors.New(string(con.Err))
		teo.log.Println(cantConnectToPeer, err)
		return
	}

	// Connect to peer
	c, err := teo.trudp.Connect(con.IP, int(con.Port))
	if err != nil {
		teo.log.Println(cantConnectToPeer, err)
		return
	}

	// Marshal peer connect request
	var conPeer ConnectToData
	conPeer.ID = con.ID
	// conPeer.Addr = teo.config.Address
	data, err = conPeer.MarshalBinary()
	if err != nil {
		teo.log.Println(cantConnectToPeer, err)
		return
	}

	// Send client peer connect request to peer
	_, err = c.Send(data)
	if err != nil {
		teo.log.Println(cantConnectToPeer, err)
		return
	}
	// TODO: Send wrong request for testing
	// _, err = c.Send(data)

	// Add teonet channel
	channel := teo.channels.new(c)
	teo.Connected(channel, con.Addr)

	return
}

// connectToConnected check received message and set connected client address
// (peer processed)
func (teo Teonet) connectToConnected(c *Channel, p *Packet) (ok bool) {
	if c.ServerMode() {
		// Teonet address example:    z6uer55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		// Teonet new(not connected): new-r55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		if p.ID() == 2 && strings.HasPrefix(c.Address(), newChannelPrefix) {

			// Unmarshal data
			var con ConnectToData
			err := con.UnmarshalBinary(p.Data)
			if err != nil {
				teo.log.Println("connectToConnected unmarshal error:", err)
				return
			}

			res, ok := teo.peerRequests.get(con.ID)
			teo.log.Println("peer request, id:", res.ID, ok, "addr:", res.Addr, "from:", c)
			if ok {
				teo.log.Println("set client connected", res.Addr, "ID:", con.ID)
				teo.Connected(c, res.Addr)
				teo.peerRequests.del(con.ID)
			} else {
				teo.channels.del(c)
				teo.log.Println("wrong request ID:", con.ID)
			}
			return true
		}
	}
	return
}

// ConnectToData teonet connect data
type ConnectToData struct {
	byteSlice
	ID        string   // Request id
	Addr      string   // Peer address
	IP        string   // Peer external ip address (sets by teonet auth)
	Port      uint32   // Peer external port (sets by teonet auth)
	LocalIPs  []string // List of local IPs (set by client or peer)
	LocalPort uint32   // Local port (set by client or peer)
	Err       []byte   // Error of connect data processing
}

func (c ConnectToData) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	c.writeSlice(buf, []byte(c.ID))
	c.writeSlice(buf, []byte(c.Addr))
	c.writeSlice(buf, []byte(c.IP))
	binary.Write(buf, binary.LittleEndian, c.Port)
	c.writeStringSlice(buf, c.LocalIPs)
	binary.Write(buf, binary.LittleEndian, c.LocalPort)
	c.writeSlice(buf, c.Err)

	data = buf.Bytes()
	return
}

func (c *ConnectToData) UnmarshalBinary(data []byte) (err error) {
	var buf = bytes.NewBuffer(data)

	if c.ID, err = c.readString(buf); err != nil {
		return
	}

	if c.Addr, err = c.readString(buf); err != nil {
		return
	}

	if c.IP, err = c.readString(buf); err != nil {
		return
	}

	if err = binary.Read(buf, binary.LittleEndian, &c.Port); err != nil {
		return
	}

	if c.LocalIPs, err = c.readStringSlice(buf); err != nil {
		return
	}

	if err = binary.Read(buf, binary.LittleEndian, &c.LocalPort); err != nil {
		return
	}

	c.Err, err = c.readSlice(buf)

	return
}
