// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Connect to teonet module

package teonet

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/kirill-scherba/bslice"
	"github.com/kirill-scherba/tru"
)

// nMODULEcon is current module name
var nMODULEcon = "Connect"

const (
	teonetReconnectAfter = 1 * time.Second
)

// Teoauth commands
const (
	// CmdConnect send <cmd byte, data ConnectData> to teonet auth server to
	// connect to teonet; receive <cmd byte, data ConnectData> from teonet auth
	// server when connection established
	CmdConnect AuthCmd = iota

	// CmdConnectTo send <cmd byte, data ConnectToData> to teonet auth server to
	// connect to peer
	CmdConnectTo

	// CmdConnectToPeer command send by teonet auth to server to receive
	// connection from client
	CmdConnectToPeer

	// CmdResendConnectTo need to resend CmdConnectTo data from rauth to auth servers
	// to find peer and send command data to it
	CmdResendConnectTo

	// CmdResendConnectToPeerto need to resend CmdConnectToPeer data from rauth
	// to auth servers to find client and send command data to it
	CmdResendConnectToPeer

	// CmdGetIP used in rauth and return channels IP:Port
	CmdGetIP
)

// AuthCmd auth command type
type AuthCmd byte

func (c AuthCmd) String() string {
	switch c {
	case CmdConnect:
		return "CmdConnect"
	case CmdConnectTo:
		return "CmdConnectTo"
	case CmdConnectToPeer:
		return "CmdConnectToPeer"
	case CmdResendConnectTo:
		return "CmdResendConnectTo"
	case CmdResendConnectToPeer:
		return "CmdResendConnectToPeer"
	case CmdGetIP:
		return "CmdGetIP"
	}
	return "not defined"
}

// Connet errors
var ErrIncorrectServerKey = errors.New("incorrect server key received")
var ErrIncorrectPublicKey = errors.New("incorrect public key received")
var ErrTimeout = errors.New("timeout")

type ConnectIpPort struct {
	IP   string
	Port int
}

type ExcludeIPs struct {
	IPs []string
}

// exclude IPs from NodeAddr slice
func (c ConnectIpPort) exclude(nodesin []NodeAddr, excludeIPs ...string) (nodes []NodeAddr) {
	nodes = nodesin
	for i := range excludeIPs {
		for j := range nodes {
			if nodes[j].IP == excludeIPs[i] {
				nodes = append(nodes[:j], nodes[j+1:]...)
				nodes = c.exclude(nodes, excludeIPs...)
				return
			}
		}
	}
	return
}

func (c *ConnectIpPort) getAddrFromHTTP(url string, excludeIPs ...string) (err error) {
	// Get connection nodes by URL
	n, err := Nodes(url)
	if err != nil {
		// log.Fatalf("can't get nodes from %s, error: %s\n", url, err)
		return
	}

	// Exclude from Nodes list by IPs
	n.address = c.exclude(n.address, excludeIPs...)

	// Return error if nodes list is empty
	l := len(n.address)
	if l == 0 {
		err = errors.New("empty list of nodes returned")
		return
	}
	fmt.Println(n)

	// Get random node
	i := 0
	if l > 1 {
		i = rand.Intn(l)
	}
	c.IP = n.address[i].IP
	c.Port = int(n.address[i].Port)
	fmt.Printf("num nodes -> %d, i -> %d, connect to: %s:%d\n", l, i, c.IP, c.Port)
	return
}

// Connect to errors

// Connect to teonet (client send request to teonet auth server):
// Client call Connect (and wait answer inside Connect function) -> Server call
// ConnectProcess -> Client got answer (inside Connect function) and set teonet
// Connected (create teonet channel)
func (teo *Teonet) Connect(attr ...interface{}) (err error) {

	teo.Log().Connect.Println(nMODULEcon, "to remote teonet node", attr)

	// Set default address if attr ommited
	if len(attr) == 0 {
		attr = append(attr, "https://teonet.cloud/auth")
	}

	// Parse attr, it may be:
	//
	//  - String with URL,
	//  - ConnectIpPort struct with IP and Port
	//  - ExcludeIPs struct with IPs slice to exclude from
	//
	// If attr string present than connect to URL by http get list of
	// available nodes remove ExludeIPs and select one of it
	var con = ConnectIpPort{"95.217.18.68", 8000}
	var excl ExcludeIPs
	var url string
	for i := range attr {
		switch v := attr[i].(type) {
		case ExcludeIPs:
			excl = v
		case ConnectIpPort:
			con = v
		case string:
			url = v
		}
	}

	// Connect to rauth https server and get auth ip:port to connect
	if len(url) > 0 {
		err = con.getAddrFromHTTP(url, excl.IPs...)
		if err != nil {
			return
		}
	}

	// Connect to tru auth node and create new teonet channel if connected
	ch, err := teo.tru.Connect(fmt.Sprintf("%s:%d", con.IP, con.Port))
	if err != nil {
		return
	}
	teo.auth = teo.channels.new(ch)

	// Create channel to wait end of connection
	var chanWait = make(chanWait)
	defer close(chanWait)

	// Subscribe to teo.auth channel to get and process messages from teonet
	// server. Subscribers reader shound return true if packet processed by this
	// reader
	var subs *subscribeData
	subs = teo.subscribe(teo.auth, func(teo *Teonet, c *Channel, p *Packet, e *Event) bool {

		// Disconnect r-host processing
		if e.Event == EventTeonetDisconnected {
			log.Debug.Printf("Connect reader got error from channel %s, error: %s\n", c, e.Err)
			teo.Unsubscribe(subs)
			teo.auth = nil
			log.Connect.Println("Disconnected from teonet")
			// Reconnect
			go func() {
				time.Sleep(100 * time.Millisecond)
				for {
					log.Debug.Println("Reconnect to teonet")
					err := teo.Connect(attr...)
					if err == nil {
						break
					}
					time.Sleep(teonetReconnectAfter)
				}
			}()
			return true
		}

		// Skip not Data Event
		if e.Event != EventData {
			return false
		}

		// Commands from teonet server processing
		cmd := teo.Command(p.Data())
		switch AuthCmd(cmd.Cmd) {

		// Client got answer to cmdConnect(connect to teonet server)
		case CmdConnect:
			// Check if chanW chanal is open
			ok := chanWait.IsOpen()
			if !ok {
				return false
			}
			// Send to channel
			chanWait <- cmd.Data

		// Client got answer to cmdConnectTo(connect to peer)
		case CmdConnectTo:
			go teo.processCmdConnectTo(cmd.Data)

		// Peer got CmdConnectToPeer command
		case CmdConnectToPeer:
			go teo.processCmdConnectToPeer(cmd.Data)

		// This commands (and empty body) added to remove "not defined" error
		// from default case
		case CmdResendConnectTo, CmdResendConnectToPeer, CmdGetIP:
			return false

		// Not defined commands
		default:
			log.Error.Println("Got not defined command", cmd.Cmd)
			return false
		}

		return true
	})
	defer func() {
		if err != nil {
			teo.Unsubscribe(subs)
		}
	}()

	// Connect data
	conIn := ConnectData{
		PubliKey:      teo.config.getPublicKey(),      // []byte("PublicKey"),
		Address:       []byte(teo.Address()),          // []byte("Address"),
		ServerKey:     teo.config.ServerPublicKeyData, // []byte("ServerKey"),
		ServerAddress: nil,
	}

	// Marshal data
	data, err := conIn.MarshalBinary()
	if err != nil {
		return
	}

	// Send to teoauth
	_, err = teo.Command(CmdConnect, data).Send(teo.auth)
	if err != nil {
		return
	}

	// Wait Connect answer data processed in subscribe callback
	select {
	case data = <-chanWait:
	case <-time.After(tru.ClientConnectTimeout):
		err = ErrTimeout
		return
	}

	// Unmarshal data
	var conOut ConnectData
	conOut.UnmarshalBinary(data)
	if err != nil {
		return
	}

	// Check server error
	if len(conOut.Err) > 0 {
		err = errors.New(string(conOut.Err))
		return
	}

	// Check received data
	if !reflect.DeepEqual(conOut.PubliKey, teo.GetPublicKey()) {
		err = ErrIncorrectPublicKey
		return
	}

	// Update config data and save config to file
	addr := string(conOut.Address)
	// teo.config.m.Lock()
	teo.config.ServerPublicKeyData = conOut.ServerKey
	teo.config.Address = addr
	// teo.config.m.Unlock()
	teo.config.save()

	teo.SetConnected(teo.auth, string(conOut.ServerAddress))

	// Connected to teonet, show log message and send Event to main reader
	log.Connect.Printf("Teonet address: %s\n", conOut.Address)
	reader(teo, teo.auth, nil, &Event{EventTeonetConnected, nil})

	return
}

// SetConnected set address to channel, add channel to channels list and send event
// SetConnected to main teonet reader
func (teo *Teonet) SetConnected(c *Channel, addr string) {
	c.a = addr
	teo.channels.add(c)
	reader(teo, c, nil, &Event{EventConnected, nil})
}

// ConnectData teonet connect data
type ConnectData struct {
	PubliKey      []byte // Client public key (generated from private key)
	Address       []byte // Client address (received after connect if empty)
	ServerKey     []byte // Server public key (send if exists or received in connect if empty)
	ServerAddress []byte // Server address (received after connect)
	Err           []byte // Error of connect data processing
	bslice.ByteSlice
}

func (c ConnectData) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	c.WriteSlice(buf, c.PubliKey)
	c.WriteSlice(buf, c.Address)
	c.WriteSlice(buf, c.ServerKey)
	c.WriteSlice(buf, c.ServerAddress)
	c.WriteSlice(buf, c.Err)

	data = buf.Bytes()
	return
}

func (c *ConnectData) UnmarshalBinary(data []byte) (err error) {

	buf := bytes.NewBuffer(data)

	c.PubliKey, err = c.ReadSlice(buf)
	if err != nil {
		return
	}
	c.Address, err = c.ReadSlice(buf)
	if err != nil {
		return
	}
	c.ServerKey, err = c.ReadSlice(buf)
	if err != nil {
		return
	}
	c.ServerAddress, err = c.ReadSlice(buf)
	if err != nil {
		return
	}
	c.Err, err = c.ReadSlice(buf)

	return
}

func (c ConnectData) String() string {
	return fmt.Sprintf("len: %d\nkey: %x\naddress: %s\nserver key: %x\nserver address: %s\nerror: %s",
		len(c.PubliKey)+len(c.Address)+len(c.ServerKey)+len(c.ServerAddress)+len(c.Err),
		c.PubliKey,
		c.Address,
		c.ServerKey,
		c.ServerAddress,
		c.Err,
	)
}
