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

	"github.com/kirill-scherba/teonet-go/teolog/teolog"
	"github.com/kirill-scherba/trudp"
)

var nMODULEconp = "Connect to peer"

const (
	peerReconnectAfter = 1 * time.Second
)

var ErrDoesNotConnectedToTeonet = errors.New("does not connected to teonet")

// ConnectTo (1) connect to any teonet Peer(client or server) by address (client
// sent request to teonet auth server):
// Client call ConnectTo wich send request to teonet auth server and wait
// function connectToAnswerProcess called -> Server call ConnectToProcess send
// infor to Peer and send answer to client (connectToAnswerProcess func called
// on client side when answer received) -> Client connect to Peer and send
// clients teonet address to it, Peer check it in connectToConnected func
func (teo Teonet) ConnectTo(addr string, readers ...interface{}) (err error) {
	// TODO: check local connection exists
	teolog.Log(teolog.CONNECT, nMODULEconp, addr)

	// Check teonet connected
	// TODO: move this code to function
	if teo.auth == nil || /* !func() bool { _, ok := teo.channels.get(teo.auth); return ok }() || */ teo.auth.IsNew() {
		err = ErrDoesNotConnectedToTeonet
		return
	}

	// Local IDs and port
	ips, _ := teo.getIPs()
	port := teo.trudp.Port()

	// Connect data
	con := ConnectToData{
		ID:        trudp.RandomString(35),
		FromAddr:  teo.MyAddr(),
		ToAddr:    addr,
		LocalIPs:  ips,
		LocalPort: uint32(port),
	}
	data, _ := con.MarshalBinary()

	// Send command to teonet
	teolog.Log(teolog.DEBUG, "Send CmdConnectTo=1 to teonet, Addr:", con.ToAddr, "ID:", con.ID)
	teo.Command(CmdConnectTo, data).Send(teo.auth)

	chanW := make(chanWait)
	defer close(chanW)
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
				teolog.Log(teolog.CONNECT, nMODULEconp, "reconnect:", c.Address())
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

// processCmdConnectToPeer (3) peer got CmdConnectToPeer request from teonet
// auth (peer prepare to connect from client and send answer with its IPs to
// auth server)
func (teo Teonet) processCmdConnectToPeer(data []byte) (err error) {

	// Unmarshal data
	var con ConnectToData
	err = con.UnmarshalBinary(data)
	if err != nil {
		teolog.Log(teolog.ERROR, "connectToPeer unmarshal error:", err)
		return
	}
	teolog.Log(teolog.DEBUG, nMODULEconp, "got CmdConnectToPeer=2 from teonet, Addr:", con.FromAddr, "ID:", con.ID)

	teo.peerRequests.add(&con)

	// Local IDs and port
	ips, _ := teo.getIPs()
	port := teo.trudp.Port()

	// Prepare answer
	conPeer := ConnectToData{
		ID:        con.ID,
		FromAddr:  con.ToAddr,
		ToAddr:    con.FromAddr,
		LocalIPs:  ips,
		LocalPort: uint32(port),
		Resend:    con.Resend,
	}
	data, _ = conPeer.MarshalBinary()

	// Send command to teonet
	teo.Command(CmdConnectToPeer, data).Send(teo.auth)

	// Punch firewall
	// TODO: calculat punch start delay here: 1) triptime from client to auth +
	// 2) triptime of this packet (from auth to this peer)
	teo.puncher.punch(con.ID, IPs{
		LocalIPs:  con.LocalIPs,
		LocalPort: con.LocalPort,
		IP:        con.IP,
		Port:      con.Port,
	}, func() bool { _, ok := teo.peerRequests.get(con.ID); return !ok },
		10*time.Millisecond,
	)

	return
}

// processCmdConnectTo check ConnectTo answer from auth server, connect to
// Peer and send clients teonet addres to it (client processed)
func (teo Teonet) processCmdConnectTo(data []byte) (err error) {

	const cantConnectToPeer = "can't connect to peer, error: "

	// Unmarshal data
	var con ConnectToData
	err = con.UnmarshalBinary(data)
	if err != nil {
		teolog.Log(teolog.ERROR, "CmdConnectTo answer unmarshal error:", err.Error())
		return
	}
	teolog.Log(teolog.DEBUG, "Got CmdConnectTo=1 answer from teonet, Addr:", con.FromAddr, "ID:", con.ID)

	// Check connRequests
	req, ok := teo.connRequests.get(con.ID)
	if !ok {
		err = errors.New("got CmdConnectTo answer time out")
		teolog.Log(teolog.ERROR, err.Error())
		return
	}

	// Check server error and send it to wait channel
	if len(con.Err) != 0 {
		// Check wait channel
		// ok := true
		// select {
		// case _, ok = <-*req.chanWait:
		// default:
		// }
		ok := req.chanWait.IsOpen()
		// Send to wait channel
		if ok {
			*req.chanWait <- con.Err
		}
		return
	}

	// Marshal peer connect request
	var conPeer ConnectToData
	conPeer.ID = con.ID
	data, err = conPeer.MarshalBinary()
	if err != nil {
		teolog.Log(teolog.ERROR, nMODULEconp, cantConnectToPeer, err)
		return
	}
	data = append([]byte(newConnectionPrefix), data...)

	// connect to peer by trudp and send it connect data
	connect := func(ip string, port uint32) (ok bool, err error) {

		_, ok = teo.connRequests.get(con.ID)
		// teo.log.Println(">>> connect to", ip, port, "skip:", !ok)
		if !ok {
			// err = errors.New("skip(already connected)")
			teolog.Log(teolog.DEBUG, nMODULEconp, "skip (already connected)")
			return
		}

		// Connect to peer
		c, err := teo.trudp.Connect(ip, int(port))
		if err != nil {
			teolog.Log(teolog.ERROR, nMODULEconp, cantConnectToPeer, err)
			return
		}

		// Send client peer connect request to peer
		teolog.Log(teolog.DEBUG, "Send answer to peer, ID:", con.ID)
		_, err = c.Send(data)
		if err != nil {
			teolog.Log(teolog.ERROR, nMODULEconp, cantConnectToPeer, err)
			return
		}

		return
	}

	// Punch firewall
	teo.puncher.punch(con.ID, IPs{
		LocalIPs:  []string{}, // empty list, don't send punch to local address
		LocalPort: con.LocalPort,
		IP:        con.IP,
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
				teolog.Log(teolog.ERROR, "CmdConnectToPeer unmarshal error:", err)
				return
			}
			teolog.Log(teolog.DEBUG, nMODULEconp, "Got answer from new client, ID:", con.ID)

			res, ok := teo.peerRequests.get(con.ID)
			if ok {
				// teo.log.Println("peer request, id:", res.ID, ok, "addr:", res.Addr, "from:", c)
				// teo.log.Println("set client connected", res.Addr, "ID:", con.ID)
				teo.Connected(c, res.FromAddr)
				teo.peerRequests.del(con.ID)
				teolog.Log(teolog.DEBUG, "Send answer to client, ID:", con.ID)
				c.SendNoWait(p.Data())
			} else {
				teo.channels.del(c)
				teolog.Log(teolog.ERROR, "wrong request ID:", con.ID)
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
				teolog.Log(teolog.ERROR, "connectToConnectedClient unmarshal error:", err)
				return
			}
			teolog.Log(teolog.DEBUG, nMODULEconp, "Got answer from new peer, ID:", con.ID)

			req, ok := teo.connRequests.get(con.ID)
			if ok {
				// teo.log.Println("got connectToConnectedClient, id:", req.ID, ok, "addr:", req.Addr, "from:", c)
				// teo.log.Println("set server connected", req.Addr, "ID:", con.ID)
				teo.Connected(c, req.ToAddr)
				// Check wait channel
				// ok := true
				// select {
				// case _, ok = <-*req.chanWait:
				// default:
				// }
				ok := req.chanWait.IsOpen()
				// Send to wait channel
				if ok {
					*req.chanWait <- nil
				}
			} else {
				teo.channels.del(c)
				teolog.Log(teolog.ERROR, "wrong request ID:", con.ID)
			}

			return true
		}
	}
	return
}

// ConnectToData teonet connect data
type ConnectToData struct {
	ByteSlice
	ID        string   // Request id
	FromAddr  string   // Peer address
	ToAddr    string   // Client address
	IP        string   // Peer external ip address (sets by teonet auth)
	Port      uint32   // Peer external port (sets by teonet auth)
	LocalIPs  []string // List of local IPs (set by client or peer)
	LocalPort uint32   // Local port (set by client or peer)
	Err       []byte   // Error of connect data processing
	Resend    bool     // Resend flag
}

func (c ConnectToData) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	c.WriteSlice(buf, []byte(c.ID))
	c.WriteSlice(buf, []byte(c.FromAddr))
	c.WriteSlice(buf, []byte(c.ToAddr))
	c.WriteSlice(buf, []byte(c.IP))
	binary.Write(buf, binary.LittleEndian, c.Port)
	c.WriteStringSlice(buf, c.LocalIPs)
	binary.Write(buf, binary.LittleEndian, c.LocalPort)
	c.WriteSlice(buf, c.Err)
	binary.Write(buf, binary.LittleEndian, c.Resend)

	data = buf.Bytes()
	return
}

func (c *ConnectToData) UnmarshalBinary(data []byte) (err error) {
	var buf = bytes.NewBuffer(data)

	if c.ID, err = c.ReadString(buf); err != nil {
		return
	}

	if c.FromAddr, err = c.ReadString(buf); err != nil {
		return
	}

	if c.ToAddr, err = c.ReadString(buf); err != nil {
		return
	}

	if c.IP, err = c.ReadString(buf); err != nil {
		return
	}

	if err = binary.Read(buf, binary.LittleEndian, &c.Port); err != nil {
		return
	}

	if c.LocalIPs, err = c.ReadStringSlice(buf); err != nil {
		return
	}

	if err = binary.Read(buf, binary.LittleEndian, &c.LocalPort); err != nil {
		return
	}

	if c.Err, err = c.ReadSlice(buf); err != nil {
		return
	}

	if err = binary.Read(buf, binary.LittleEndian, &c.Resend); err != nil {
		return
	}

	return
}
