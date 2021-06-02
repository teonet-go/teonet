// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Connect to teonet module

package teonet

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/kirill-scherba/trudp"
)

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
)

// AuthCmd auth command type
type AuthCmd byte

// Connet errors
var ErrIncorrectServerKey = errors.New("incorrect server key received")
var ErrIncorrectPublicKey = errors.New("incorrect public key received")
var ErrTimeout = errors.New("timeout")

type ConnectIpPort struct {
	IP   string
	Port int
}

func (c *ConnectIpPort) getAddrFromHTTP(url string) (err error) {
	n, err := Nodes(url)
	if err != nil {
		// log.Fatalf("can't get nodes from %s, error: %s\n", url, err)
		return
	}
	if len(n.address) == 0 {
		err = errors.New("empty list of nodes returned")
		return
	}
	fmt.Println(n)
	// TODO: get i from random
	i := 0
	c.IP = n.address[i].IP
	c.Port = int(n.address[i].Port)
	return
}

// Connect to errors

// Connect to teonet (client send request to teonet auth server):
// Client call Connect (and wait answer inside Connect function) -> Server call
// ConnectProcess -> Client got answer (inside Connect function) and set teonet
// Connected (create teonet channel)
func (teo *Teonet) Connect(attr ...interface{}) (err error) {

	teo.log.Println("connect to teonet")

	// Set default address if attr ommited
	if len(attr) == 0 {
		attr = append(attr, "http://teonet.kekalan.cloud:10000/auth")
	}

	var con = ConnectIpPort{"95.217.18.68", 8000}
	for i := range attr {
		switch v := attr[i].(type) {
		case ConnectIpPort:
			con = v
		case string:
			err = con.getAddrFromHTTP(v)
			if err != nil {
				return
			}
		}
	}

	// TODO: Connect to auth https server and get auth ip:port to connect to
	//

	// Connect to trudp auth node
	ch, err := teo.trudp.Connect(con.IP, con.Port)
	if err != nil {
		return
	}

	var subs *subscribeData
	var chanW = make(chan []byte)
	defer close(chanW)
	teo.auth = teo.channels.new(ch)
	// Subscribe to teo.auth channel to get and process messages from teonet
	// server. Subscribers reader shound return true if packet processed by this
	// reader
	subs = teo.subscribe(teo.auth, func(teo *Teonet, c *Channel, p *Packet, err error) bool {

		// Error processing
		if err != nil {
			teo.log.Printf("connect reader: got error from channel %s, error: %s", c, err)
			teo.unsubscribe(subs)
			teo.log.Println("disconnected from teonet")
			// Reconnect
			go func() {
				for {
					err := teo.Connect(attr...)
					if err == nil {
						break
					}
					time.Sleep(teonetReconnectAfter)
				}
			}()
			return true
		}

		// Commands from teonet server processing
		cmd := teo.Command(p.Data())
		switch AuthCmd(cmd.Cmd) {

		// Client got answer to cmdConnect(connect to teonet server)
		case CmdConnect:
			// Check if chanW chanal is open
			ok := true
			select {
			case _, ok = <-chanW:
			default:
			}
			// Send to channel
			if !ok {
				return false
			}
			chanW <- cmd.Data

		// Client got answer to cmdConnectTo(connect to peer)
		case CmdConnectTo:
			go teo.connectToAnswerProcess(cmd.Data)

		// Peer got CmdConnectToPeer command
		case CmdConnectToPeer:
			go teo.connectToPeer(cmd.Data)

		// Not defined commands
		default:
			teo.log.Println("not defined command", cmd.Cmd)
			return false
		}

		return true
	})

	// Connect data
	conIn := ConnectData{
		PubliKey:      teo.config.getPublicKey(),      // []byte("PublicKey"),
		Address:       []byte(teo.config.Address),     // []byte("Address"),
		ServerKey:     teo.config.ServerPublicKeyData, // []byte("ServerKey"),
		ServerAddress: nil,
	}

	// Marshal data
	data, err := conIn.MarshalBinary()
	if err != nil {
		// teo.log.Println("encode error:", err)
		return
	}
	// teo.log.Println("encoded ConnectData:", data, len(data))

	// Send to teoauth
	// cmd := teo.Command(CmdConnect, data)
	// _, err = teo.trudp.Send(teo.auth.c, cmd.Bytes())
	_, err = teo.Command(CmdConnect, data).Send(teo.auth)
	if err != nil {
		return
	}
	// teo.log.Println("send ConnectData to teoauth, id", id)

	// Wait Connect answer data
	select {
	case data = <-chanW:
	case <-time.After(trudp.ClientConnectTimeout):
		err = ErrTimeout
		teo.unsubscribe(subs)
		return
	}

	// Unmarshal data
	var conOut ConnectData
	conOut.UnmarshalBinary(data)
	if err != nil {
		// teo.log.Println("decode error:", err)
		return
	}
	// teo.log.Printf("decoded ConnectData: %s\n", conOut)

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

	// Update config file
	addr := string(conOut.Address)
	teo.config.ServerPublicKeyData = conOut.ServerKey
	teo.config.Address = addr
	teo.config.save()

	teo.log.Println("connected to teonet")
	teo.log.Printf("teonet address: %s\n", conOut.Address)

	teo.Connected(teo.auth, string(conOut.ServerAddress))

	return
}

// Connected set address to channel and add channel to channels list
func (teo Teonet) Connected(c *Channel, addr string) {
	c.a = addr
	teo.channels.add(c)
}

// ConnectData teonet connect data
type ConnectData struct {
	byteSlice
	PubliKey      []byte // Client public key (generated from private key)
	Address       []byte // Client address (received after connect if empty)
	ServerKey     []byte // Server public key (send if exists or received in connect if empty)
	ServerAddress []byte // Server address (received after connect)
	Err           []byte // Error of connect data processing
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
