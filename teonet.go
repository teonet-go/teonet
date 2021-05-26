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

	"github.com/kirill-scherba/trudp"
)

const Version = "0.0.1"

// Logo print teonet logo
func Logo(title, ver string) {
	fmt.Println("" +
		" _____                     _   \n" +
		"|_   _|__  ___  _ __   ___| |_ \n" +
		"  | |/ _ \\/ _ \\| '_ \\ / _ \\ __|\n" +
		"  | |  __/ (_) | | | |  __/ |_ \n" +
		"  |_|\\___|\\___/|_| |_|\\___|\\__|\n" +
		"\n" +
		title + " ver " + ver + ", based on teonet ver " + Version +
		"\n",
	)
}

// Log get teonet log to use it in application and inside teonet
func Log() *log.Logger {
	return log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds /* |log.Lshortfile */)
}

// reader is Main teonet reader
func reader(teo *Teonet, c *Channel, p *Packet, err error) {

	// Check error and 'connect to peer connected' processing
	if err != nil {
		teo.channels.del(c)
	} else if teo.connectToConnected(c, p) {
		return
	}

	// Send to subscribers readers (to readers from teo.subscribe)
	if teo.subscribers.send(teo, c, p, err) {
		return
	}

	// Send to client reader (to reader from teonet.Init)
	if teo.clientReader != nil {
		teo.clientReader(teo, c, p, err)
	}
}

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
		log       *log.Logger
		reader    Treceivecb
	}
	for i := range attr {
		switch d := attr[i].(type) {
		case int:
			param.port = d
		case string:
			param.logLevel = d
		case bool:
			param.showTrudp = d
		case *log.Logger:
			param.log = d
		// case Treceivecb:
		case func(teo *Teonet, c *Channel, p *Packet, err error) bool:
			param.reader = d
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
	teo.log = log

	// Create config holder and read config
	err = teo.newConfig(appName, log)
	if err != nil {
		return
	}

	// Init trudp and start listen port to get messages
	teo.clientReader = param.reader
	teo.trudp, err = trudp.Init(param.port, teo.config.trudpPrivateKey, teo.log, param.logLevel,

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
				pac = &Packet{p} // &Packet{p.Header, p.Data}
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
				time.Sleep(3 * time.Second)
				_, exists := teo.channels.get(c)
				if !exists {
					teo.log.Println("remove dummy trudp channel:", c)
					c.ChannelDel(c)
				}
			}(c)
		},
	)
	if err != nil {
		teo.log.Println("can't initial trudp, error:", err)
		return
	}
	teo.newChannels()
	teo.log.Println("start listen at port", teo.trudp.Port())

	if param.showTrudp {
		teo.ShowTrudp(true)
	}

	return
}

type Teonet struct {
	config       *config
	trudp        *trudp.Trudp
	log          *log.Logger
	clientReader Treceivecb
	subscribers  *subscribersData
	channels     *channels
	auth         *Channel
}

// ShowTrudp show/stop trudp statistic
func (teo Teonet) ShowTrudp(set bool) {
	teo.trudp.SetShowStat(set)
}

// Send data to peer
func (teo Teonet) SendTo(addr string, data []byte) (id uint32, err error) {
	c, ok := teo.channels.get(addr)
	if !ok {
		err = errors.New("peer does not connected")
		return
	}
	return c.Send(data)
}
