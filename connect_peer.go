// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Connect to peer module

package teonet

import (
	"bytes"
	"encoding/binary"
	"errors"
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

	// Connect data
	conIn := ConnectToData{
		Addr: addr,
	}
	data, _ := conIn.MarshalBinary()
	// Send command to teonet
	teo.Command(CmdConnectTo, data).Send(teo.auth)
	teo.log.Println("send CmdConnectTo", addr, "(send to teo.auth)")

	return
}

// ConnectToProcess check connection to teonet peer and send answer to client
// and request to Peer (auth server receive connection data from client)
func (teo Teonet) ConnectToProcess(c *Channel, data []byte) (err error) {

	// Unmarshal data
	var con ConnectToData
	con.UnmarshalBinary(data)
	if err != nil {
		teo.log.Println("decode error:", err)
		return
	}
	teo.log.Println("got CmdConnectTo", con.Addr, "from", c)

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
		var conSer ConnectToData
		// Client address, IPs and port
		conSer.Addr = c.a
		conSer.IP = c.c.UDPAddr.IP.String()
		conSer.Port = uint32(c.c.UDPAddr.Port)
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
func (teo Teonet) connectToPeer(data []byte) {
	teo.log.Println("got CmdConnectToPeer command, data len:", len(data))
}

// connectToAnswerProcess check ConnectTo answer from auth server, connect to
// Peer and send clients teonet addres to it (client processed)
func (teo Teonet) connectToAnswerProcess(data []byte) (err error) {

	const cantConnectToPeer = "can't connect to peer, error: "

	// Unmarshal data
	var con ConnectToData
	con.UnmarshalBinary(data)
	if err != nil {
		teo.log.Println("unmarshal error:", err)
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

	// Send client teonet address to peer
	c.Send([]byte(teo.config.Address))

	// Add teonet channel
	channel := teo.channels.new(c)
	teo.Connected(channel, con.Addr)

	return
}

// connectToConnected check received message and set connected client address
// (peer processed)
func (teo Teonet) connectToConnected(c *Channel, p *Packet) (ok bool) {
	if c.ServerMode() {
		// z6uer55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		if p.ID() == 2 && c.Address() == "" && len(p.Data) == 35 {
			teo.log.Println("set client connected", c.c)
			teo.Connected(c, string(p.Data))
			return
		}
	}
	return
}

// ConnectToData teonet connect data
type ConnectToData struct {
	byteSlice
	Addr string // Peer address
	IP   string // Peer ip address
	Port uint32 // Peer port
	Err  []byte // Error of connect data processing
}

func (c ConnectToData) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	c.writeSlice(buf, []byte(c.Addr))
	c.writeSlice(buf, []byte(c.IP))
	err = binary.Write(buf, binary.LittleEndian, c.Port)
	c.writeSlice(buf, c.Err)

	data = buf.Bytes()
	return
}

func (c *ConnectToData) UnmarshalBinary(data []byte) (err error) {
	buf := bytes.NewBuffer(data)

	d, err := c.readSlice(buf)
	if err != nil {
		return
	}
	c.Addr = string(d)

	d, err = c.readSlice(buf)
	if err != nil {
		return
	}
	c.IP = string(d)

	err = binary.Read(buf, binary.LittleEndian, &c.Port)
	if err != nil {
		return
	}

	c.Err, err = c.readSlice(buf)

	return
}
