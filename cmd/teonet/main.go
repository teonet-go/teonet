package main

import (
	"flag"
	"os"
	"time"

	"github.com/kirill-scherba/teonet"
)

const (
	appName    = "Teonet sample application"
	appShort   = "teonet"
	appVersion = "0.0.4"
)

// reader main application reade receive and process messages
func reader(teo *teonet.Teonet, c *teonet.Channel, p *teonet.Packet, err error) bool {
	// Check errors
	if err != nil {
		// teolog.Println("channel", c, "read error:", err)
		return false
	}

	// Print received message
	teo.Log().Printf("got from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n",
		c, p.Data(), len(p.Data()), p.ID(),
		float64(c.Triptime().Microseconds())/1000.0,
	)

	// Send answer in server mode
	if c.ServerMode() {
		answer := []byte("Teonet answer to " + string(p.Data()))
		c.SendAnswer(answer)
	}

	return true
}

func main() {

	// Application logo
	teonet.Logo(appName, appVersion)

	// Parse applications flags
	var params struct {
		appShort    string
		showTrudp   bool
		showPrivate bool
		sendTo      string
		logLevel    string
	}
	flag.StringVar(&params.appShort, "app-short", appShort, "application short name")
	flag.BoolVar(&params.showTrudp, "u", false, "show trudp statistic")
	flag.BoolVar(&params.showPrivate, "show-private", false, "show private key")
	flag.StringVar(&params.sendTo, "send-to", "", "send messages to address")
	flag.StringVar(&params.logLevel, "log-level", "NONE", "log level")
	flag.Parse()

	// Start teonet client
	teo, err := teonet.New(params.appShort, 0, reader, teonet.Log(), "NONE", params.showTrudp, params.logLevel)
	if err != nil {
		teo.Log().Println("can't init Teonet, error:", err)
		return
	}

	// Show this application private key
	if params.showPrivate {
		teo.Log().Printf("%x\n", teo.GetPrivateKey())
		os.Exit(0)
	}

	// Connect to teonet
	err = teo.Connect()
	if err != nil {
		teo.Log().Println("can't connect to Teonet, error:", err)
		return
	}

	// Connect to Peer (selected in send-to application flag) and receive
	// packets in own reader
	if params.sendTo != "" {
		err := teo.ConnectTo(params.sendTo,
			// Receive and process packets from this channel(address). Return
			// true if packet processed. If return false package will processed
			// by other readers include main application reader (just comment
			// 'processed = true' line and you'll see two 'got from ...' message)
			func(c *teonet.Channel, p *teonet.Packet, err error) (processed bool) {
				if err == nil {
					// Print received message
					teo.Log().Printf("got(r) from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n",
						c, p.Data(), len(p.Data()), p.ID(), float64(c.Triptime().Microseconds())/1000.0,
					)
					processed = true
				}
				return
			},
		)
		if err != nil {
			teo.Log().Println("can't connect to Peer, error:", err)
		}
	}

	// Send to Peer
	if params.sendTo != "" {
		for {
			time.Sleep(5 * time.Second)
			_, err = teo.SendTo(params.sendTo, []byte("Hello world!"))
			if err != nil {
				teo.Log().Println(err)
				continue
			}
			teo.Log().Println("send message to", params.sendTo)
		}
	}

	select {} // sleep forever
}
