// Teonet client connected to teonet server with api interface sample application
package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/kirill-scherba/teonet"
)

const (
	appShort   = "teoapicli"
	appName    = "Teonet api client sample application"
	appVersion = "0.0.1"
)

func main() {

	// Application logo
	teonet.Logo(appName, appVersion)

	// Parse applications flags
	var params struct {
		appShort  string
		port      int
		showTrudp bool
		logLevel  string
		logFilter string
		connectTo string
	}
	flag.StringVar(&params.appShort, "app-short", appShort, "application short name")
	flag.IntVar(&params.port, "p", 0, "local port")
	flag.BoolVar(&params.showTrudp, "u", false, "show trudp statistic")
	flag.StringVar(&params.connectTo, "connect-to", "", "connect to api server")
	flag.StringVar(&params.logLevel, "log-level", "NONE", "log level")
	flag.StringVar(&params.logFilter, "log-filter", "", "log filter")
	flag.Parse()

	if params.connectTo == "" {
		fmt.Println("Flag -log-level should be set")
		flag.Usage()
		return
	}

	// Start teonet (client or server)
	teo, err := teonet.New(params.appShort, params.port, params.showTrudp,
		params.logLevel, teonet.LogFilterT(params.logFilter))
	if err != nil {
		teo.Log().Println("can't init Teonet, error:", err)
		return
	}

	teo.Log().Println("Start")

	// Connect to teonet
	for teo.Connect() != nil {
		// teo.Log().Println("can't connect to Teonet, error:", err)
		time.Sleep(1 * time.Second)
	}

	// Teonet address
	fmt.Printf("Teonet addres: %s\n\n", teo.Address())

	// Connect to API server (selected in connect-to application flag) and receive
	// packets in own reader. Use WXJfYLDEtg6Rkm1OHm9I9ud9rR6qPlMH6NE addres to
	// connect to installed teoapi example.
	for {
		err := teo.ConnectTo(params.connectTo,
			// Receive and process packets from this channel(address). Return
			// true if packet processed. If return false package will processed
			// by other readers include main application reader (just comment
			// 'processed = true' line and you'll see two 'got from ...' message)
			func(c *teonet.Channel, p *teonet.Packet, err error) (processed bool) {
				if err == nil {
					// Print received message
					// teo.Log().Printf("got(r) from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n\n",
					// 	c, p.Data(), len(p.Data()), p.ID(), float64(c.Triptime().Microseconds())/1000.0,
					// )
					teo.Log().Printf("Got '%s', from %s\n", p.Data(), c)
					processed = true
				}
				return
			},
		)
		if err == nil {
			break
		}
		teo.Log().Println("can't connect to API server, error:", err)
		time.Sleep(1 * time.Second)
	}

	// Check API server commands
	teo.Log().Printf("Connected to API server: %s\n\n", params.connectTo)

	// Send command 129('hello')
	data := []byte("Kirill")
	teo.Log().Println("Send 129('hello') with data:", string(data))
	teo.Command(129, []byte("Kirill")).SendTo(params.connectTo)

	// Send command 130('description')
	teo.Log().Println("Send 130('description') without data")
	teo.Command(130, nil).SendTo(params.connectTo)

	// teo.Log().Println()

	select {} // sleep forever
}
