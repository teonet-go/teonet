// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Connect to teonet module

package teonet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/kirill-scherba/trudp"
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

type AuthCmd byte

// Connet errors
var ErrIncorrectServerKey = errors.New("incorrect server key received")
var ErrIncorrectPublicKey = errors.New("incorrect public key received")
var ErrTimeout = errors.New("timeout")

// Connect to errors

// Connect to teonet (client send request to teonet auth server):
// Client call Connect (and wait answer inside Connect function) -> Server call
// ConnectProcess -> Client got answer (inside Connect function) and set teonet
// Connected (create teonet channel)
func (teo *Teonet) Connect(auth ...string) (err error) {

	teo.log.Println("connect to teonet")

	// TODO: Connect to auth https server and get auth ip:port to connect to
	//

	// Connect to trudp auth node
	ch, err := teo.trudp.Connect("localhost", 8000)
	if err != nil {
		return
	}

	var subs *subscribeData
	var chanW = make(chan []byte)
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
					err := teo.Connect(auth...)
					if err == nil {
						break
					}
					time.Sleep(1 * time.Second)
				}
			}()
			return true
		}

		// Commands from teonet server processing
		cmd := teo.Command(p.Data)
		switch AuthCmd(cmd.Cmd) {

		// Client got answer to cmdConnect(connect to teonet server)
		case CmdConnect:
			chanW <- cmd.Data

		// Client got answer to cmdConnectTo(connect to peer)
		case CmdConnectTo:
			go teo.connectToAnswerProcess(cmd.Data)

		// Peer got CmdConnectToPeer command
		case CmdConnectToPeer:
			teo.connectToPeer(cmd.Data)

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

// ConnectProcess check connection to teonet and send answer to client (auth
// server receive connection data from client)
func (teo Teonet) ConnectProcess(c *Channel, data []byte) (err error) {
	// Unmarshal data
	var con ConnectData
	con.UnmarshalBinary(data)
	if err != nil {
		teo.log.Println("decode error:", err)
		return
	}
	// teo.log.Printf("decoded ConnectData: %s\n", con)

	sendAnswer := func() (err error) {
		// Encode
		data, err = con.MarshalBinary()
		if err != nil {
			teo.log.Println("marshal error:", err)
			return
		}

		// Send snswer
		// cmd := teo.Command(CmdConnect, data)
		// _, err = c.c.SendAnswer(cmd.Bytes())
		_, err = teo.Command(CmdConnect, data).SendAnswer(c)
		if err != nil {
			teo.log.Println("send answer error:", err)
		}
		return
	}

	// Check server key and set it if empty
	if len(con.ServerKey) == 0 {
		con.ServerKey = teo.GetPublicKey()
	} else if !reflect.DeepEqual(con.ServerKey, teo.GetPublicKey()) {
		con.Err = []byte(ErrIncorrectServerKey.Error())
		err = sendAnswer()
		return
	}

	// Set server Address
	con.ServerAddress = []byte(teo.GetAddress())

	// TODO: check client Address
	if len(con.Address) == 0 {
		var addr string
		addr, err = teo.MakeAddress(con.PubliKey)
		if err != nil {
			teo.log.Println("make client address error:", err)
			return
		}
		con.Address = []byte(addr)
	}

	err = sendAnswer()
	if err != nil {
		return
	}

	// Add to clients map
	teo.Connected(c, string(con.Address))
	// teo.log.Println("client connected:", string(con.Address))

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

	c.writeSlice(buf, c.PubliKey)
	c.writeSlice(buf, c.Address)
	c.writeSlice(buf, c.ServerKey)
	c.writeSlice(buf, c.ServerAddress)
	c.writeSlice(buf, c.Err)

	data = buf.Bytes()
	return
}

func (c *ConnectData) UnmarshalBinary(data []byte) (err error) {

	buf := bytes.NewBuffer(data)

	c.PubliKey, err = c.readSlice(buf)
	if err != nil {
		return
	}
	c.Address, err = c.readSlice(buf)
	if err != nil {
		return
	}
	c.ServerKey, err = c.readSlice(buf)
	if err != nil {
		return
	}
	c.ServerAddress, err = c.readSlice(buf)
	if err != nil {
		return
	}
	c.Err, err = c.readSlice(buf)

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

type byteSlice struct{}

func (b byteSlice) writeSlice(buf *bytes.Buffer, data []byte) (err error) {
	if err = binary.Write(buf, binary.LittleEndian, uint16(len(data))); err != nil {
		return
	}
	err = binary.Write(buf, binary.LittleEndian, data)
	return
}

func (b byteSlice) readSlice(buf *bytes.Buffer) (data []byte, err error) {
	var l uint16
	if err = binary.Read(buf, binary.LittleEndian, &l); err != nil {
		return
	}
	data = make([]byte, l)
	err = binary.Read(buf, binary.LittleEndian, data)
	return
}

func (b byteSlice) readString(buf *bytes.Buffer) (data string, err error) {
	d, err := b.readSlice(buf)
	if err != nil {
		return
	}
	data = string(d)
	return
}

func (b byteSlice) writeStringSlice(buf *bytes.Buffer, data []string) (err error) {
	if err = binary.Write(buf, binary.LittleEndian, uint16(len(data))); err != nil {
		return
	}
	for i := range data {
		if err = b.writeSlice(buf, []byte(data[i])); err != nil {
			return
		}
	}

	return
}

func (b byteSlice) readStringSlice(buf *bytes.Buffer) (data []string, err error) {
	var l uint16
	if err = binary.Read(buf, binary.LittleEndian, &l); err != nil {
		return
	}
	for i := 0; i < int(l); i++ {
		var d []byte
		if d, err = b.readSlice(buf); err != nil {
			return
		}
		data = append(data, string(d))
	}
	return
}
