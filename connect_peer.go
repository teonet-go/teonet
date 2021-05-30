// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Connect to peer module

package teonet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"time"

	"github.com/kirill-scherba/trudp"
)

const (
	peerReconnectAfter = 1 * time.Second
)

// ConnectTo connect to any teonet Peer(client or server) by address (client
// sent request to teonet auth server):
// Client call ConnectTo wich send request to teonet auth server and wait
// function connectToAnswerProcess called -> Server call ConnectToProcess send
// infor to Peer and send answer to client (connectToAnswerProcess func called
// on client side when answer received) -> Client connect to Peer and send
// clients teonet address to it, Peer check it in connectToConnected func
func (teo Teonet) ConnectTo(addr string, readers ...interface{}) (err error) {
	// TODO: check local connection exists

	// Local IDs and port
	ips, _ := teo.getIPs()
	port := teo.trudp.Port()

	// Connect data
	con := ConnectToData{
		ID:        trudp.RandomString(35),
		Addr:      addr,
		LocalIPs:  ips,
		LocalPort: uint32(port),
	}
	data, _ := con.MarshalBinary()

	// Send command to teonet
	teo.Command(CmdConnectTo, data).Send(teo.auth)
	// teo.log.Println("send CmdConnectTo", con.Addr, "ID:", con.ID)

	chanW := make(chanWait)
	teo.connRequests.add(&con, &chanW)
	defer teo.connRequests.del(con.ID)

	// Wait Connect answer data
	select {
	case d := <-chanW:
		if len(d) > 0 {
			err = errors.New(string(d))
			return
		}
	case <-time.After(trudp.ClientConnectTimeout):
		err = ErrTimeout
		return
	}

	// Connected, make auto reconnect
	teo.Subscribe(addr, func(teo *Teonet, c *Channel, p *Packet, err error) (ret bool) {
		if err != nil {
			go func() {
				teo.Log().Println("reconnect:", c.Address())
				for {
					err := teo.ConnectTo(addr, readers...)
					if err == nil {
						break
					}
					time.Sleep(peerReconnectAfter)
				}
			}()
		}
		return
	})

	// Subscribe to channel
	for i := range readers {
		teo.Subscribe(addr, readers[i])
	}

	return
}

// connectToPeer peer got ConnectTo request from teonet auth (peer prepare to
// connect from client and send answer with it IPs to auth server)
func (teo Teonet) connectToPeer(data []byte) (err error) {

	// Unmarshal data
	var con ConnectToData
	err = con.UnmarshalBinary(data)
	if err != nil {
		teo.log.Println("connectToPeer unmarshal error:", err)
		return
	}
	// teo.log.Println("got CmdConnectToPeer command", con.Addr, "ID:", con.ID, con)
	teo.peerRequests.add(&con)

	// Local IDs and port
	ips, _ := teo.getIPs()
	port := teo.trudp.Port()

	// Prepare answer
	conPeer := ConnectToData{
		ID:        con.ID,
		Addr:      con.Addr,
		LocalIPs:  ips,
		LocalPort: uint32(port),
	}
	data, _ = conPeer.MarshalBinary()

	// Send command to teonet
	teo.Command(CmdConnectToPeer, data).Send(teo.auth)

	// Punch firewall
	teo.puncher.punch(con.ID, IPs{
		LocalIPs:  con.LocalIPs,
		LocalPort: con.LocalPort,
		IP:        con.ID,
		Port:      con.Port,
	}, func() bool { _, ok := teo.peerRequests.get(con.ID); return !ok },
	/* 100*time.Millisecond, */
	)

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
	// teo.log.Println("got CmdConnectTo answer:", con.Addr, con)

	// Check connRequests
	req, ok := teo.connRequests.get(con.ID)
	if !ok {
		teo.log.Println("got CmdConnectTo connection time out")
		return
	}

	// Check server error and send it to wait channel
	if len(con.Err) != 0 {
		*req.chanWait <- con.Err
		return
	}

	// Marshal peer connect request
	var conPeer ConnectToData
	conPeer.ID = con.ID
	data, err = conPeer.MarshalBinary()
	if err != nil {
		teo.log.Println(cantConnectToPeer, err)
		return
	}
	data = append([]byte(newConnectionPrefix), data...)

	// connect to peer
	connect := func(ip string, port uint32) (ok bool, err error) {

		_, ok = teo.connRequests.get(con.ID)
		// teo.log.Println(">>> connect to", ip, port, "skip:", !ok)
		if !ok {
			// err = errors.New("skip(already connected)")
			return
		}

		// Connect to peer
		c, err := teo.trudp.Connect(ip, int(port))
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

		return
	}

	// Punch firewall
	teo.puncher.punch(con.ID, IPs{
		LocalIPs:  []string{}, // con.LocalIPs, // empty list of local address
		LocalPort: con.LocalPort,
		IP:        con.ID,
		Port:      con.Port,
	}, func() bool { _, ok := teo.connRequests.get(con.ID); return !ok })

	waitCh := make(chan *net.UDPAddr)
	teo.puncher.subscribe(con.ID, &PuncherData{&waitCh})

	// TODO: add timeout here
	addr := <-waitCh

	connect(addr.IP.String(), uint32(addr.Port))

	return
}

// connectToConnectedPeer check received message from client, set connected client
// address and send answer (peer processed)
func (teo Teonet) connectToConnectedPeer(c *Channel, p *Packet) (ok bool) {
	if c.ServerMode() {
		// Teonet address example:    z6uer55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		// Teonet new(not connected): new-r55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		if p.ID() == 2 && c.IsNew() && c.IsConn(p.Data()) {

			// Unmarshal data
			var con ConnectToData
			err := con.UnmarshalBinary(p.Data()[len(newConnectionPrefix):])
			if err != nil {
				teo.log.Println("connectToConnected unmarshal error:", err)
				return
			}

			res, ok := teo.peerRequests.get(con.ID)
			if ok {
				// teo.log.Println("peer request, id:", res.ID, ok, "addr:", res.Addr, "from:", c)
				// teo.log.Println("set client connected", res.Addr, "ID:", con.ID)
				teo.Connected(c, res.Addr)
				teo.peerRequests.del(con.ID)
				c.SendAnswer(p.Data())
			} else {
				teo.channels.del(c)
				teo.log.Println("wrong request ID:", con.ID)
			}
			return true
		}
	}
	return
}

// connectToConnectedClient check received message and set connected peer
// address (client processed)
func (teo Teonet) connectToConnectedClient(c *Channel, p *Packet) (ok bool) {
	if c.ClientMode() {
		// Teonet address example:    z6uer55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		// Teonet new(not connected): new-r55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		if p.ID() == 1 && c.IsNew() && c.IsConn(p.Data()) {

			// Unmarshal data
			var con ConnectToData
			err := con.UnmarshalBinary(p.Data()[len(newConnectionPrefix):])
			if err != nil {
				teo.log.Println("connectToConnectedClient unmarshal error:", err)
				return
			}

			req, ok := teo.connRequests.get(con.ID)
			if ok {
				// teo.log.Println("got connectToConnectedClient, id:", req.ID, ok, "addr:", req.Addr, "from:", c)
				// teo.log.Println("set server connected", req.Addr, "ID:", con.ID)
				teo.Connected(c, req.Addr)
				*req.chanWait <- nil
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
