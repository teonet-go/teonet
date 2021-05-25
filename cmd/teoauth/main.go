package main

import (
	"flag"

	"github.com/kirill-scherba/teonet"
)

const (
	appName    = "Teonet auth application"
	appShort   = "teoauth"
	appVersion = "0.0.1"
)

var teolog = teonet.Log()

// reader receive and process messages
func reader(teo *teonet.Teonet, c *teonet.Channel, p *teonet.Packet, err error) bool {
	// Check errors
	if err != nil {
		// teolog.Println("channel", c, "read error:", err)
		return true
	}

	// Print received message
	// teolog.Printf("got from %s, \"%s\", len: %d, tt: %6.3fms\n",
	// 	c, p.Data, len(p.Data), float64(c.Triptime().Microseconds())/1000.0,
	// )

	// Send answer in server mode
	if c.ServerMode() {
		err := teo.ConnectProcess(c, p.Data)
		if err != nil {
			teolog.Println("connect process error:", err)
			return true
		}
	}
	return true
}

func main() {
	teonet.Logo(appName, appVersion)

	var params struct {
		appShort  string
		showTrudp bool
	}
	flag.StringVar(&params.appShort, "app-short", appShort, "application short name")
	flag.BoolVar(&params.showTrudp, "u", false, "show trudp statistic")
	flag.Parse()

	_, err := teonet.New(params.appShort, 8000, reader, teolog, "NONE", params.showTrudp)
	if err != nil {
		teolog.Println("can't init Teonet, error:", err)
		return
	}

	// teolog.Println("done")

	select {} // sleep forever
}
