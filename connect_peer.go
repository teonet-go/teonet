// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet connect to peer module

package teonet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/kirill-scherba/bslice"
	"github.com/teonet-go/tru"
)

var nMODULEconp = "connectto"

const peerReconnectAfter = 1 * time.Second

var ErrDoesNotConnectedToTeonet = errors.New("does not connected to teonet")
var ErrPeerDoesNotExists = errors.New("peer does not exists")

// ConnectTo connect to any teonet Peer(client or server) by address
func (teo Teonet) ConnectTo(addr string, readers ...interface{}) (err error) {

	// During ConnectTo client sent request to Teonet auth server:
	//
	//   - Client call ConnectTo which send request to teonet auth server and wait
	//   function connectToAnswerProcess called
	//
	//   - Server call ConnectToProcess send infor to Peer and send answer to
	//   client (connectToAnswerProcess func called on client side when answer
	//   received)
	//
	//   - Client connect to Peer and send clients teonet address to it, Peer check
	//   it in connectToConnected func
	//

	log.Connect.Println(nMODULEconp, addr)

	// Check teonet connected
	var auth = teo.getAuth()
	if auth == nil || auth.IsNew() {
		err = ErrDoesNotConnectedToTeonet
		return
	}

	// Check peer already connected
	_, ok := teo.channels.get(addr)
	if ok {
		return
	}

	// Local IPs and port
	ips, _ := teo.getIPs()
	port := teo.tru.LocalPort()

	// Connect data
	con := ConnectToData{
		ID:        tru.RandomString(35),
		FromAddr:  teo.Address(),
		ToAddr:    addr,
		LocalIPs:  ips,
		LocalPort: uint32(port),
	}
	data, _ := con.MarshalBinary()

	// Send command to teonet
	log.Debugv.Println(nMODULEconp, "send request to teonet, addr:",
		con.ToAddr[:8], "id:", con.ID[:8])
	teo.Command(CmdConnectTo, data).Send(auth)

	// Wait and receive punch answer
	teo.clientPunchReceive(&con)

	// Create wait channel and connect request
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
	case <-time.After(tru.ClientConnectTimeout):
		err = ErrTimeout
		return
	}

	// Connected, make auto reconnect
	var scr *subscribeData
	scr, _ = teo.Subscribe(addr, func(teo *Teonet, c *Channel, p *Packet, e *Event) (ret bool) {
		// Peer disconnected event
		if e.Event == EventDisconnected {
			// Unsubscribe
			teo.Unsubscribe(scr)

			select {
			// Return if teonet closing
			case <-teo.closing:
				return
			default:
				// Return if channel closing
				if c.closing {
					return
				}
				// Reconnect to disconnected peer
				go func() {
					for {
						log.Connect.Println(nMODULEconp, "reconnect:", c.Address())
						err := teo.ConnectTo(addr, readers...)
						if err == nil {
							break
						}
						time.Sleep(peerReconnectAfter)
					}
				}()
			}
		}
		return
	})

	// Subscribe to channel
	for i := range readers {
		teo.Subscribe(addr, readers[i])
	}

	return
}

// CloseTo close connection to peere previously opened by ConnecTo
func (teo Teonet) CloseTo(addr string) (err error) {
	log.Debug.Println("close connection to peer", addr)
	ch, ok := teo.channels.get(addr)
	if !ok {
		err = ErrPeerDoesNotExists
		return
	}
	ch.closing = true
	ch.c.Close()
	return
}

// ReconnectOff will stop reconnection when peer will be disconnected. By
// default all Teonet connections will forewer try automatic reconnect when
// peer disoconnected. To stop this reconnection call ReconnectOff any time
// after ConnetTo.
func (teo Teonet) ReconnectOff(addr string) (err error) {
	log.Debug.Println("stop reconnection to peer", addr)
	ch, ok := teo.channels.get(addr)
	if !ok {
		err = ErrPeerDoesNotExists
		return
	}
	ch.closing = true
	return
}

// WhenConnectedTo call faunction f when connected to peer by address
func (teo *Teonet) WhenConnectedTo(address string, f func()) {
	teo.clientReaders.addShort(func(c *Channel, p *Packet, e *Event) (processed bool) {
		if e.Event == EventConnected && c.Address() == address {
			f()
		}
		return
	})
}

// WhenConnectedDisconnected call faunction f when connected or disconnected
// to any peer
func (teo *Teonet) WhenConnectedDisconnected(f func(e byte)) {
	teo.clientReaders.addShort(func(c *Channel, p *Packet, ev *Event) (processed bool) {
		switch ev.Event {
		case EventConnected, EventDisconnected:
			f(byte(ev.Event))
		}
		return
	})
}

// processCmdConnectToPeer process CmdConnectToPeer request from teonet
// auth (peer prepare to connect from client and send answer with its IPs to
// auth server) (server processwd)
func (teo Teonet) processCmdConnectToPeer(data []byte) (err error) {

	// Check teonet connected
	var auth = teo.getAuth()
	if auth == nil || auth.IsNew() {
		err = ErrDoesNotConnectedToTeonet
		return
	}

	// Unmarshal data
	var con = new(ConnectToData)
	err = con.UnmarshalBinary(data)
	if err != nil {
		log.Error.Println(nMODULEconp,
			"got request from teonet unmarshal error:", err)
		return
	}
	log.Debugv.Println(nMODULEconp,
		"got request from teonet",
		"addr:", con.FromAddr[:6],
		"id:", con.ID[:6],
		// "from ip:", con.IP+":"+strconv.Itoa(int(con.Port)),
	)

	teo.peerRequests.add(con)

	// Local IDs and port
	ips, _ := teo.getIPs()
	port := teo.tru.LocalPort()

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
	teo.Command(CmdConnectToPeer, data).Send(auth)

	// Punch firewall
	teo.serverPunchReceive(con)
	teo.serverPunchSend(con)

	return
}

// Subscribe to puncher answer - set wait channel and punch firewall (server)
func (teo Teonet) serverPunchReceive(con *ConnectToData) {

	// Subscribe to punch messages and wait answer from puncher or timeout
	waitCh := make(chan *net.UDPAddr)
	teo.puncher.subscribe(con.ID, &PuncherData{&waitCh})
	go func() {
		select {

		// Answer received
		case addr := <-waitCh:
			// When punch received (by server) resend it data back to sender
			teo.log.Debugv.Println("send answer to puncher message to", addr.String())
			teo.puncher.send(con.ID, IPs{
				IP:   addr.IP.String(),
				Port: uint32(addr.Port),
			})

		// Timeout
		case <-time.After(tru.ClientConnectTimeout):
			teo.puncher.unsubscribe(con.ID)

		}
	}()
}

// serverPunchSend send clients punch
func (teo Teonet) serverPunchSend(con *ConnectToData) {
	go func() {
		// Punch firewall (from server to client - server mode)
		teo.puncher.punch(con.ID, IPs{
			LocalIPs:  con.LocalIPs,
			LocalPort: con.LocalPort,
			IP:        con.IP,
			Port:      con.Port,
		}, func() bool { _, ok := teo.peerRequests.get(con.ID); return !ok })
	}()
}

// processCmdConnectTo process ConnectTo answer from auth server, connect to
// Peer and send clients teonet address to it (client processed)
func (teo Teonet) processCmdConnectTo(data []byte) (err error) {

	// Unmarshal data
	var con = new(ConnectToData)
	err = con.UnmarshalBinary(data)
	if err != nil {
		log.Error.Println(nMODULEconp, "got responce from teonet unmarshal error:", err.Error())
		return
	}
	log.Debugv.Println(nMODULEconp, "got responce from teonet,",
		"addr:", con.FromAddr[:8],
		"id:", con.ID[:8],
		"ip", con.IP+":"+strconv.Itoa(int(con.Port)),
	)

	// Check connRequests
	req, ok := teo.connRequests.get(con.ID)
	if !ok {
		err = errors.New("got responce from teonet, already processed or time out")
		log.Debugv.Println(nMODULEconp, err.Error())
		return
	}

	// Check server error and send it to wait channel
	if len(con.Err) != 0 {
		// Check wait channel is open and send to wait channel if opened
		if req.chanWait.IsOpen() {
			*req.chanWait <- con.Err
		}
		return
	}

	// Punch firewall
	// teo.clientPunchReceive(con)
	teo.clientPunchSend(con)

	return
}

// clientPunchReceive subscribe to puncher answer and wait servers punch messages
func (teo Teonet) clientPunchReceive(con *ConnectToData) {

	const cantConnectToPeer = "can't connect to peer, error: "

	// connect to peer by tru and send it connect data
	connect := func(ip string, port int) (ok bool, err error) {

		_, ok = teo.connRequests.get(con.ID)
		if !ok {
			log.Debug.Println(nMODULEconp, "skip (already connected)")
			return
		}

		// Connect to peer
		ip, _ = teo.safeIPv6(ip)
		c, err := teo.tru.Connect(fmt.Sprintf("%s:%d", ip, port))
		if err != nil {
			log.Error.Println(nMODULEconp, cantConnectToPeer, err)
			return
		}

		// Marshal peer connect request
		var conPeer ConnectToData
		conPeer.ID = con.ID
		data, err := conPeer.MarshalBinary()
		if err != nil {
			log.Error.Println(nMODULEconp, cantConnectToPeer, err)
			return
		}
		data = append([]byte(newConnectionPrefix), data...)

		// Send client peer connect request to peer
		log.Debugv.Println(nMODULEconp, "send request to peer, id:", con.ID[:8])
		_, err = c.WriteTo(data)
		if err != nil {
			log.Error.Println(nMODULEconp, cantConnectToPeer, err)
		}

		return
	}

	// Subscribe to punch messages and wait answer from puncher or timeout
	waitCh := make(chan *net.UDPAddr)
	teo.puncher.subscribe(con.ID, &PuncherData{&waitCh})
	go func() {

		var err error

		// Wait answer from puncher or timeout
		select {

		// Answer received
		case addr := <-waitCh:
			_, err = connect(addr.IP.String(), addr.Port)

		// Timeout
		case <-time.After(tru.ClientConnectTimeout):
			teo.puncher.unsubscribe(con.ID)
			err = ErrTimeout
		}

		if err != nil {
			log.Debugv.Println("can't punch during connect, err", err)
		}
	}()
}

// clientPunchSend send clients punch
func (teo Teonet) clientPunchSend(con *ConnectToData) {
	go func() {
		// Punch firewall (from client to server - client mode)
		teo.puncher.punch(con.ID, IPs{
			LocalIPs:  []string{}, // empty list, don't send punch to local address
			LocalPort: con.LocalPort,
			IP:        con.IP,
			Port:      con.Port,
		}, func() bool { _, ok := teo.connRequests.get(con.ID); return !ok })
	}()
}

// connectToPeer check received message from client, set connected client
// address and send answer (peer processed)
func (teo Teonet) connectToPeer(c *Channel, p *Packet) (ok bool) {
	if c.ServerMode() {
		// Teonet address example:    z6uer55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		// Teonet new(not connected): new-r55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		if p.ID() == 0 && c.IsNew() && c.IsConn(p.Data()) {

			// Unmarshal data
			var con ConnectToData
			err := con.UnmarshalBinary(p.Data()[len(newConnectionPrefix):])
			if err != nil {
				log.Error.Println(nMODULEconp, "CmdConnectToPeer unmarshal error:", err)
				return
			}
			log.Debugv.Println(nMODULEconp, "got answer from new client, id:", con.ID[:6])

			if res, ok := teo.peerRequests.del(con.ID); ok {
				// Set channel connected
				teo.SetConnected(c, res.FromAddr)
				// Send answer to client
				log.Debugv.Println(nMODULEconp, "send answer to client, id:", con.ID[:6])
				c.Send(p.Data())
			} else {
				log.Error.Println(nMODULEconp, "!!! wrong request id:", con.ID[:6])
				// TODO: we can't delete channel here becaus deadlock will be
				// Check if we need delete, and what hapend if we does not delete
				// teo.channels.del(c)
			}
			return true
		}
	}
	return
}

// connectToClient check received message and set connected peer
// address (client processed)
func (teo Teonet) connectToClient(c *Channel, p *Packet) (ok bool) {
	if c.ClientMode() {
		// Teonet address example:    z6uer55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		// Teonet new(not connected): new-r55DZsqvY5pqXHjTD3oDFfsKmkfFJ65
		if p.ID() == 0 && c.IsNew() && c.IsConn(p.Data()) {

			// Unmarshal data
			var con ConnectToData
			err := con.UnmarshalBinary(p.Data()[len(newConnectionPrefix):])
			if err != nil {
				log.Error.Println(nMODULEconp,
					"got responce from peer unmarshal error:", err)
				return
			}
			log.Debugv.Println(nMODULEconp, "got responce from peer, id:",
				con.ID[:8])

			if req, ok := teo.connRequests.get(con.ID); ok {
				// Set channel connected
				teo.SetConnected(c, req.ToAddr)
				// Send to wait channel to finish connection and close connRequest
				if req.chanWait.IsOpen() {
					*req.chanWait <- nil
				}
			} else {
				log.Error.Println(nMODULEconp, "!!! wrong request id:", con.ID)
				// TODO: thr same question as in previuse func
				// teo.channels.del(c)
			}
			return true
		}
	}
	return
}

// ConnectToData teonet connect data
type ConnectToData struct {
	ID        string   // Request id
	FromAddr  string   // Peer address
	ToAddr    string   // Client address
	IP        string   // Peer external ip address (sets by teonet auth)
	Port      uint32   // Peer external port (sets by teonet auth)
	LocalIPs  []string // List of local IPs (set by client or peer)
	LocalPort uint32   // Local port (set by client or peer)
	Err       []byte   // Error of connectTo processing
	Resend    bool     // Resend flag
	bslice.ByteSlice
}

// MarshalBinary binary marshal ConnectToData structure
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

// UnmarshalBinary binary unmarshal ConnectToData structure
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
