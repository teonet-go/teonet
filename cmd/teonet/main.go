package main

import (
	"flag"
	"os"

	"github.com/kirill-scherba/teonet"
)

const (
	appName    = "Teonet sample application"
	appShort   = "teonet"
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
	teolog.Printf("got from %s, \"%s\", len: %d, tt: %6.3fms\n",
		c, p.Data, len(p.Data), float64(c.Triptime().Microseconds())/1000.0,
	)

	// Send answer in server mode
	if c.ServerMode() {
		answer := []byte("Teonet answer to " + string(p.Data))
		c.SendAnswer(answer)
	}
	return true
}

func main() {
	teonet.Logo(appName, appVersion)

	var params struct {
		appShort    string
		showTrudp   bool
		showPrivate bool
	}
	flag.StringVar(&params.appShort, "app-short", appShort, "application short name")
	flag.BoolVar(&params.showTrudp, "u", false, "show trudp statistic")
	flag.BoolVar(&params.showPrivate, "show-private", false, "show private key")
	flag.Parse()

	teo, err := teonet.New(params.appShort, 0, reader, teolog, "NONE", params.showTrudp)
	if err != nil {
		teolog.Println("can't init Teonet, error:", err)
		return
	}

	if params.showPrivate {
		teolog.Printf("%x\n", teo.GetPrivateKey())
		os.Exit(0)
	}

	err = teo.Connect()
	if err != nil {
		teolog.Println("can't connect to Teonet, error:", err)
		return
	}

	select {} // sleep forever in server mode
}
