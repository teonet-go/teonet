// Teonet echo client/server sample application
package main

import (
	"flag"
	"os"
	"time"

	"github.com/kirill-scherba/teomon/teomon"
	"github.com/kirill-scherba/teonet"
)

const (
	appName    = "Teonet echo client/server sample application"
	appShort   = "teoecho"
	appVersion = "0.2.9"

	// Teonet Monitor address
	monitor = "nOhj2qRDKduN9sHIRoRmJ3LTjOfrKey8llq"
)

var appStartTime = time.Now()

// reader is main application reader it receive and process messages
func reader(teo *teonet.Teonet, c *teonet.Channel, p *teonet.Packet, e *teonet.Event) bool {
	// Skip not Data events
	// if err != nil {
	if e.Event != teonet.EventData {
		return false
	}

	// In server mode
	if c.ServerMode() {

		// Print received message
		teo.Log().Printf("got from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n",
			c, p.Data(), len(p.Data()), p.ID(),
			float64(c.Triptime().Microseconds())/1000.0,
		)

		// Send answer
		answer := []byte("Teonet answer to " + string(p.Data()))
		c.SendNoWait(answer)
	}

	return true
}

func main() {

	// Application logo
	teonet.Logo(appName, appVersion)

	// Parse applications flags
	var params struct {
		appShort    string
		port        int
		showTrudp   bool
		showPrivate bool
		sendTo      string
		logLevel    string
		logFilter   string
	}
	flag.StringVar(&params.appShort, "app-short", appShort, "application short name")
	flag.IntVar(&params.port, "p", 0, "local port")
	flag.BoolVar(&params.showTrudp, "u", false, "show trudp statistic")
	flag.BoolVar(&params.showPrivate, "show-private", false, "show private key")
	flag.StringVar(&params.sendTo, "send-to", "", "send messages to address")
	flag.StringVar(&params.logLevel, "log-level", "NONE", "log level")
	flag.StringVar(&params.logFilter, "log-filter", "", "log filter")
	flag.Parse()

	// Start teonet client
	teo, err := teonet.New(params.appShort, params.port, reader, teonet.Log(), "NONE",
		params.showTrudp, params.logLevel, teonet.LogFilterT(params.logFilter),
	)
	if err != nil {
		teo.Log().Println("can't init Teonet, error:", err)
		return
	}

	// Show this application private key
	if params.showPrivate {
		teo.Log().Printf("%x\n", teo.GetPrivateKey())
		os.Exit(0)
	}

connect:
	// Connect to teonet
	err = teo.Connect()
	if err != nil {
		teo.Log().Println("can't connect to Teonet, error:", err)
		time.Sleep(1 * time.Second)
		goto connect
	}

	// Sleep forever if sendTo flag does not set (in server mode)
	if params.sendTo == "" {

		// Connect to monitor
		teomon.Connect(teo, monitor, teomon.Metric{
			AppName:      appName,
			AppShort:     appShort,
			AppVersion:   appVersion,
			TeoVersion:   teonet.Version,
			AppStartTime: appStartTime,
		})

		select {}
	}

connectto:
	// Connect to Peer (selected in send-to application flag) and receive
	// packets in own reader
	if err := teo.ConnectTo(params.sendTo,
		// Receive and process packets from this channel(address). Return
		// true if packet processed. If return false package will processed
		// by other readers include main application reader (just comment
		// 'processed = true' line and you'll see two 'got from ...' message)
		func(c *teonet.Channel, p *teonet.Packet, e *teonet.Event) (processed bool) {

			// Skip not Data Events
			if e.Event != teonet.EventData {
				return
			}

			// Print received message
			teo.Log().Printf("got(r) from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n\n",
				c, p.Data(), len(p.Data()), p.ID(), float64(c.Triptime().Microseconds())/1000.0,
			)
			processed = true

			return
		},
	); err != nil {
		teo.Log().Println("can't connect to Peer, error:", err)
		time.Sleep(1 * time.Second)
		goto connectto
	}

sendto:
	// Send to Peer
	time.Sleep(5 * time.Second)
	teo.Log().Println("send message to", params.sendTo)
	if _, err := teo.SendTo(params.sendTo, []byte("Hello world!")); err != nil {
		teo.Log().Println(err)
	}
	goto sendto
}
