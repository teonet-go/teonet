// Teonet server with teonet api sample application
package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/kirill-scherba/teonet"
)

const (
	appName    = "Teonet api server sample application"
	appShort   = "teoapi"
	appVersion = "0.0.1"
)

func Commands(teo *teonet.Teonet, api *teonet.API) {

	api.Add(
		teonet.MakeAPI(
			"hello",
			"<name string> get hello name message",
			"usage: hello <name string>",
			129,
			teonet.Server,
			func(c *teonet.Channel, data []byte) bool {
				data = append([]byte("Hello "), data...)
				// Use SendNoWait function when you answer to just received
				// command. If processing of you command get lot of time (read
				// data from data base or read file etc.) do it in goroutine
				// and use Send() function. If you don't shure which to use
				// than use Send() function :)
				c.SendNoWait(data)
				return true
			}),
		teonet.MakeAPI(
			"description",
			"got application description",
			"usage: description",
			130,
			teonet.Server,
			func(c *teonet.Channel, data []byte) bool {
				ret := []byte(appName)
				c.SendNoWait(ret)
				return true
			}),
	)
}

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

	// Start teonet (client or server)
	teo, err := teonet.New(params.appShort, params.port, params.showTrudp,
		params.logLevel, teonet.LogFilterT(params.logFilter))
	if err != nil {
		teo.Log().Println("can't init Teonet, error:", err)
		return
	}

	api := teonet.NewAPI(teo)
	Commands(teo, api)
	teo.AddReader(api.Reader())

	// Print API
	fmt.Printf("API description:\n%s\n\n", api)

	// Connect to teonet
	for teo.Connect() != nil {
		// teo.Log().Println("can't connect to Teonet, error:", err)
		time.Sleep(1 * time.Second)
	}

	// Teonet address
	fmt.Printf("Teonet addres: %s\n\n", teo.Address())

	select {} // sleep forever
}
