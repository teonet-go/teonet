// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet v4

package teonet

import (
	"errors"
	"fmt"
	"time"

	"github.com/kirill-scherba/tru"
	"github.com/kirill-scherba/tru/teolog"
)

const Version = "0.3.0"

// nMODULEteo is current module name
var nMODULEteo = "Teonet"

var log *teolog.Teolog

// Log get teonet log to use it in application and inside teonet
func Log() *teolog.Teolog {
	return log
}

// Logo print teonet logo
func Logo(title, ver string) {
	fmt.Println(LogoString(title, ver))
}

func LogoString(title, ver string) string {
	return fmt.Sprint("" +
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

// reader is Main teonet reader
func reader(teo *Teonet, c *Channel, p *Packet, e *Event) {

	// Delete channel on err after all other reader process this error
	// TODO: Realy need this defer?
	defer func() {
		if e.Err != nil {
			teo.channels.del(c, false)
		}
	}()

	// Check error and 'connect to peer connected' processing
	// if e.Err == nil && (teo.connectToConnectedPeer(c, p) || teo.connectToConnectedClient(c, p)) {
	if e.Event == EventData && (teo.connectToConnectedPeer(c, p) || teo.connectToConnectedClient(c, p)) {
		return
	}

	// Send to subscribers readers (to readers from teo.subscribe)
	if teo.subscribers.send(teo, c, p, e) {
		return
	}

	// Send to client readers (to reader from teonet.Init)
	teo.clientReaders.send(teo, c, p, e)
}

type LogFilter = teolog.Filter
type Stat = tru.Stat
type Hotkey = tru.Hotkey

func Logfilter(str string) teolog.Filter { return teolog.Logfilter(str) }

// New create new teonet connection. The attr parameters:
//   int             port number to teonet listen
//   string          internal log Level to show teonet debug messages
//   tru.ShowStat    set true to show tru statistic table
//   tru.StartHotkey start hotkey meny
//   *teolog.Teolog  teonet logger
//   ApiInterface    api interface
//   OsConfigDir     os directory to save config
//   func(c *Channel, p *Packet, e *Event) - message receiver
//   func(t *Teonet, c *Channel, p *Packet, e *Event) - message receiver
func New(appName string, attr ...interface{}) (teo *Teonet, err error) {

	// Parse attributes
	var param struct {
		port      int
		stat      tru.Stat
		hotkey    tru.Hotkey
		logLevel  string
		logFilter LogFilter
		log       *teolog.Teolog
		reader    Treceivecb
		api       ApiInterface
		configDir OsConfigDir
	}
	for i := range attr {
		switch d := attr[i].(type) {
		// Local port
		case int:
			param.port = d
		// Log level
		case string:
			param.logLevel = d
		// Log filter
		case teolog.Filter:
			param.logFilter = d
		// Show tru statistic flag
		case tru.Stat:
			param.stat = d
		// Start hotkey menu
		case tru.Hotkey:
			param.hotkey = d
		// Logger
		case *teolog.Teolog:
			param.log = d
		// Treceivecb:
		case func(teo *Teonet, c *Channel, p *Packet, e *Event) bool:
			param.reader = d
		// TreceivecbShort:
		case func(c *Channel, p *Packet, e *Event) bool:
			param.reader = func(t *Teonet, c *Channel, p *Packet, e *Event) bool {
				return d(c, p, e)
			}
		// API interface
		case ApiInterface:
			param.api = d
		// Config file folder
		case OsConfigDir:
			param.configDir = d
		// Some enother (incorrect) attribute
		default:
			err = fmt.Errorf("incorrect attribute type '%T'", d)
			return
		}
	}
	if param.logLevel == "" {
		param.logLevel = "NONE"
	}
	if param.log == nil {
		param.log = teolog.New()
	}
	log = param.log

	// Create new teonet holder
	teo = new(Teonet)
	teo.newSubscribers()
	teo.newPeerRequests()
	teo.newConnRequests()
	teo.newClientReaders()
	teo.log = log

	// Create config holder and read config
	err = teo.newConfig(appName, string(param.configDir))
	if err != nil {
		return
	}

	// Add client readers
	teo.addApiReader(param.api)
	teo.clientReaders.add(param.reader)

	// Init tru and start listen port to get messages
	teo.tru, err = tru.New(param.port, teo.log, param.stat, param.hotkey,
		param.logLevel, param.logFilter,
		teo.config.trudpPrivateKey,

		// Receive data callback
		// ch *tru.Channel, pac *tru.Packet, err error
		func(c *tru.Channel, p *tru.Packet, err error) bool {
			ch, ok := teo.channels.get(c)
			if !ok {
				if teo.auth != nil && c == teo.auth.c {
					ch = teo.auth
				} else {
					ch = teo.channels.new(c)
				}
			}

			// Create packet
			var pac *Packet
			if p != nil {
				pac = &Packet{p, ch.a, false}
			}

			// Create Disconnect, TeonetDisconnect or Data Events
			e := new(Event)
			if err != nil {
				e.Err = err
				if ch == teo.auth {
					e.Event = EventTeonetDisconnected
				} else {
					e.Event = EventDisconnected
				}
			} else {
				e.Event = EventData
			}

			// Send packet and event to main teonet reader
			// TODO: add return bool to reader func or not add :-)
			reader(teo, ch, pac, e)
			return true
		},

		// Connect to this server callback
		func(c *tru.Channel, err error) {
			// Wait this tru channel connected to teonet channel and delete
			// it if not connected during timeout
			_, exists := teo.channels.get(c)
			if exists {
				return
			}
			go func(c *tru.Channel) {
				time.Sleep(tru.ClientConnectTimeout)
				ch, exists := teo.channels.get(c)
				if !exists {
					if newch, ok := teo.channels.getByIP(c.Addr().String()); !ok {
						log.Debug.Println(nMODULEteo, "remove dummy tru channel:", c, ch)
						c.Close()
					} else {
						log.Debugvv.Println(nMODULEteo, "tru channel was reconnected:", c.Addr().String(), newch)
					}
				} else if ch.IsNew() {
					log.Debug.Println(nMODULEteo, "remove dummy(new) teonet channel:", c, ch)
					teo.channels.del(ch)
				}
			}(c)
		},
	)
	if err != nil {
		log.Error.Println(nMODULEteo, "can't initial tru, error:", err)
		return
	}
	teo.newChannels()
	teo.newPuncher()
	log.Connect.Println(nMODULEteo, "start listen teonet at port", teo.tru.LocalPort())

	return
}

type Teonet struct {
	config        *config
	tru           *tru.Tru
	log           *teolog.Teolog
	clientReaders *clientReaders
	subscribers   *subscribers
	channels      *channels
	auth          *Channel
	peerRequests  *connectRequests
	connRequests  *connectRequests
	puncher       *puncher
}

type Treceivecb func(teo *Teonet, c *Channel, p *Packet, e *Event) bool
type TreceivecbShort func(c *Channel, p *Packet, e *Event) bool

type Event struct {
	Event TeonetEventType
	Err   error
}

type TeonetEventType byte

// Teonet events
const (
	EventNone TeonetEventType = iota

	// Event when Teonet client initialized and start listen, Err = nil
	EventTeonetInit

	// Event when Connect to teonet r-host, Err = nil
	EventTeonetConnected

	// Event when Disconnect from teonet r-host, Err = dosconnect error
	EventTeonetDisconnected

	// Event when Connect to peer, Err = nil
	EventConnected

	// Event when Disconnect from peer, Err = dosconnect error
	EventDisconnected

	// Event when Data Received, Err = nil
	EventData
)

func (e Event) String() (str string) {
	switch e.Event {
	case EventNone:
		str = "EventNone"
	case EventTeonetInit:
		str = "EventTeonetInit"
	case EventTeonetConnected:
		str = "EventTeonetConnected"
	case EventTeonetDisconnected:
		str = "EventTeonetDisconnected"
	case EventConnected:
		str = "EventConnected"
	case EventDisconnected:
		str = "EventDisconnected"
	case EventData:
		str = "EventData"
	}
	return
}

func (teo Teonet) Rhost() *Channel { return teo.auth }

// ShowTrudp show/stop tru statistic
func (teo Teonet) ShowTrudp(set bool) {
	if set {
		teo.tru.StatisticPrint()
	} else {
		teo.tru.StatisticPrintStop()
	}
}

var ErrPeerNotConnected = errors.New("peer does not connected")

// Send data to peer
func (teo *Teonet) SendTo(addr string, data []byte, attr ...interface{}) (id int, err error) {
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
	return c.c.WriteTo(data, attr...)
}

// Connected return true if peer with selected address is connected now
func (teo *Teonet) Connected(addr string) (ok bool) {
	_, ok = teo.channels.get(addr)
	return
}

// Log get teonet log
func (teo Teonet) Log() *teolog.Teolog {
	return teo.log
}

// Port get teonet local port
func (teo Teonet) Port() int {
	return teo.tru.LocalPort()
}
