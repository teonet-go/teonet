// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet v4

package teonet

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kirill-scherba/teonet-go/teolog/teolog"
	"github.com/kirill-scherba/trudp"
)

const Version = "0.1.2"

// nMODULEteo is current module name
var nMODULEteo = "Teonet"

// Logo print teonet logo
func Logo(title, ver string) {
	fmt.Println("" +
		" _____                     _   \n" +
		"|_   _|__  ___  _ __   ___| |_  v4\n" +
		"  | |/ _ \\/ _ \\| '_ \\ / _ \\ __|\n" +
		"  | |  __/ (_) | | | |  __/ |_ \n" +
		"  |_|\\___|\\___/|_| |_|\\___|\\__|\n" +
		"\n" +
		title + " ver " + ver + ", based on Teonet v4 ver " + Version +
		"\n",
	)
}

// Log get teonet log to use it in application and inside teonet
func Log() *log.Logger {
	return log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds /* |log.Lshortfile */)
}

// reader is Main teonet reader
func reader(teo *Teonet, c *Channel, p *Packet, err error) {

	// Delete channel on err after all other reader process this error
	defer func() {
		if err != nil {
			teo.channels.del(c)

		}
	}()

	// Check error and 'connect to peer connected' processing
	if err == nil && (teo.connectToConnectedPeer(c, p) || teo.connectToConnectedClient(c, p)) {
		return
	}

	// Send to subscribers readers (to readers from teo.subscribe)
	if teo.subscribers.send(teo, c, p, err) {
		return
	}

	// Send to client readers (to reader from teonet.Init)
	for i := range teo.clientReaders {
		if teo.clientReaders[i] != nil {
			if teo.clientReaders[i](teo, c, p, err) {
				break
			}
		}
	}
}

type LogFilterT = trudp.LogFilterT

// New create new teonet connection. The attr parameters:
//   int - port number to teonet listen
//   string - internal log Level to show teonet debug messages
//   bool - set true to show trudp statistic table
//   *log.Logger - common logger to show messages in application and teonet, may be created with teonet.Log() function
//   func(c *Channel, p *Packet, err error) - message receiver
func New(appName string, attr ...interface{}) (teo *Teonet, err error) {

	// Parse attributes
	var param struct {
		port      int
		showTrudp bool
		logLevel  string
		logFilter LogFilterT
		log       *log.Logger
		reader    Treceivecb
		api       ApiInterface
	}
	for i := range attr {
		switch d := attr[i].(type) {
		case int:
			param.port = d
		case string:
			param.logLevel = d
		case trudp.LogFilterT:
			param.logFilter = d
		case bool:
			param.showTrudp = d
		case *log.Logger:
			param.log = d
		// case Treceivecb:
		case func(teo *Teonet, c *Channel, p *Packet, err error) bool:
			param.reader = d
		// case TreceivecbShort:
		case func(c *Channel, p *Packet, err error) bool:
			param.reader = func(t *Teonet, c *Channel, p *Packet, err error) bool {
				return d(c, p, err)
			}
		case ApiInterface:
			fmt.Println("set api")
			param.api = d
		default:
			err = fmt.Errorf("wrong attribute type %T", d)
			return
		}
	}
	if param.logLevel == "" {
		param.logLevel = "NONE"
	}
	if param.log == nil {
		param.log = Log()
	}
	log := param.log

	// Create new teonet holder
	teo = new(Teonet)
	teo.newSubscribers()
	teo.newPeerRequests()
	teo.newConnRequests()
	teo.log = log

	// Create config holder and read config
	err = teo.newConfig(appName, log)
	if err != nil {
		return
	}

	// Init trudp and start listen port to get messages
	teo.addClientReader(param.reader)
	teo.setApiReader(param.api)
	teo.trudp, err = trudp.Init(param.port, teo.config.trudpPrivateKey, teo.log,
		param.logLevel, trudp.LogFilterT(param.logFilter),

		// Receive data callback
		func(c *trudp.Channel, p *trudp.Packet, err error) {
			ch, ok := teo.channels.get(c)
			if !ok {
				if teo.auth != nil && c == teo.auth.c {
					ch = teo.auth
					// teo.log.Println("!!! auth channel !!! ", c)
				} else {
					ch = teo.channels.new(c)
					// teo.log.Println("!!! new empty channel !!! ", c)
				}
			}
			// else {
			// 	// teo.log.Println("!!! exists channel !!! ", c)
			// }
			var pac *Packet
			if p != nil {
				pac = &Packet{p, ch.a, false}
			}
			reader(teo, ch, pac, err)
		},

		// Connect to this server callback
		func(c *trudp.Channel, err error) {
			// Wait this trudp channel connected to teonet channel and delete
			// it if not connected during timeout
			_, exists := teo.channels.get(c)
			// teo.log.Println("server connection done, from", c.String(), "error:", err,
			// 	"teonet channel exists:", exists)
			if exists {
				return
			}
			go func(c *trudp.Channel) {
				time.Sleep(trudp.ClientConnectTimeout)
				ch, exists := teo.channels.get(c)
				if !exists /* || ch.IsNew() */ {
					// fmt.Println("c.String()", c.String())
					if newch, ok := teo.channels.getByIP(c.String()); !ok {
						teolog.Log(teolog.DEBUG, nMODULEteo, "remove dummy trudp channel:", c, ch)
						c.ChannelDel(c)
					} else {
						teolog.Log(teolog.DEBUGvv, nMODULEteo, "trudp channel was reconnected:", c.String(), newch)
					}
				} else if ch.IsNew() {
					teolog.Log(teolog.DEBUG, nMODULEteo, "remove dummy(new) teonet channel:", c, ch)
					teo.channels.del(ch)
				}
			}(c)
		},
	)
	if err != nil {
		teolog.Log(teolog.ERROR, nMODULEteo, "can't initial trudp, error:", err)
		return
	}
	teo.newChannels()
	teo.newPuncher()
	teolog.Log(teolog.CONNECT, nMODULEteo, "start listen teonet at port", teo.trudp.Port())

	if param.showTrudp {
		teo.ShowTrudp(true)
	}

	return
}

type Teonet struct {
	config        *config
	trudp         *trudp.Trudp
	log           *log.Logger
	clientReaders []Treceivecb
	subscribers   *subscribers
	channels      *channels
	auth          *Channel
	peerRequests  *connectRequests
	connRequests  *connectRequests
	puncher       *puncher
}

type Treceivecb func(teo *Teonet, c *Channel, p *Packet, err error) bool
type TreceivecbShort func(c *Channel, p *Packet, err error) bool

func (teo Teonet) Rhost() *Channel { return teo.auth }

// addClientReader add teonet client reader
func (teo *Teonet) addClientReader(reader Treceivecb) {
	teo.clientReaders = append(teo.clientReaders, reader)
}

// AddReader add teonet client reader
func (teo *Teonet) AddReader(reader TreceivecbShort) {
	teo.clientReaders = append(teo.clientReaders, func(teo *Teonet, c *Channel, p *Packet, err error) bool {
		return reader(c, p, err)
	})
}

// ShowTrudp show/stop trudp statistic
func (teo Teonet) ShowTrudp(set bool) {
	teo.trudp.SetShowStat(set)
}

var ErrPeerNotConnected = errors.New("peer does not connected")

// Send data to peer
func (teo *Teonet) SendTo(addr string, data []byte, attr ...interface{}) (id uint32, err error) {
	// Check address
	c, ok := teo.channels.get(addr)
	if !ok {
		err = ErrPeerNotConnected
		return
	}
	// Add teo to attr, it need for subscribe to answer
	if len(attr) > 0 {
		attr = append([]interface{}{teo}, attr...)
	}
	// Send to channel
	return c.Send(data, attr...)
}

// Log get teonet log
func (teo Teonet) Log() *log.Logger {
	return teo.log
}

// Port get teonet local port
func (teo Teonet) Port() uint32 {
	return uint32(teo.trudp.Port())
}

// Get this app Address
func (teo Teonet) MyAddr() string {
	return teo.config.Address
}
