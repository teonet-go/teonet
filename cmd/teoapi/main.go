// Teonet server with teonet api sample application
package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/kirill-scherba/teomon/teomon"
	"github.com/kirill-scherba/teonet"
)

const (
	appName    = "Teonet api server sample application"
	appShort   = "teoapi"
	appVersion = "0.2.8"
	appLong    = ""

	// Teonet Monitor address
	monitor = "nOhj2qRDKduN9sHIRoRmJ3LTjOfrKey8llq"
)

func Commands(teo *teonet.Teonet, api *teonet.API) {

	api.Add(
		func(cmdApi teonet.APInterface) teonet.APInterface {
			cmdApi = teonet.MakeAPI2().
				SetCmd(api.Cmd(129)).                 // Command number cmd = 129
				SetName("hello").                     // Command name
				SetShort("get 'hello name' message"). // Short description
				SetUsage("<name string>").            // Usage (input parameter)
				SetReturn("<answer string>").         // Return (output parameters)
				// Command reader (execute when command received)
				SetReader(func(c *teonet.Channel, p *teonet.Packet, data []byte) bool {
					data = append([]byte("Hello "), data...)
					api.SendAnswer(cmdApi, c, data, p)
					return true
				}).SetAnswerMode( /* teonet.CmdAnswer | */ teonet.DataAnswer)
			return cmdApi
		}(teonet.APIData{}),

		func(cmdApi teonet.APInterface) teonet.APInterface {
			cmdApi = teonet.MakeAPI2().
				SetCmd(api.CmdNext()).                   // Command number cmd = 130
				SetName("description").                  // Command name
				SetShort("get application description"). // Short description
				SetReturn("<description string>").       // Return (output parameters)
				// Command reader (execute when command received)
				SetReader(func(c *teonet.Channel, p *teonet.Packet, data []byte) bool {
					ret := []byte(appName)
					api.SendAnswer(cmdApi, c, ret, p)
					return true
				})
			return cmdApi
		}(teonet.APIData{}),

		func(cmdApi teonet.APInterface) teonet.APInterface {
			cmdApi = teonet.MakeAPI2().
				SetCmd(api.CmdNext()).      // Command number cmd = 131
				SetName("secret").          // Command name
				SetShort("get secret key"). // Short description
				SetUsage("<id string>").    // Usage (input parameter)
				SetReturn("<secret data>"). // Return (output parameters)
				// Command reader (execute when command received)
				SetReader(func(c *teonet.Channel, p *teonet.Packet, data []byte) bool {
					ret := []byte("this is very strong secret key: ququruqu")
					api.SendAnswer(cmdApi, c, ret, p)
					return true
				}).SetAnswerMode( /* teonet.CmdAnswer | */ teonet.PacketIDAnswer)
			return cmdApi
		}(teonet.APIData{}),
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

	// Create new API, add commands and reader
	api := teo.NewAPI(appName, appShort, appLong, appVersion)
	Commands(teo, api)
	teo.AddReader(api.Reader())

	// Print API
	fmt.Printf("API description:\n\n%s\n\n", api.Help())

	// Connect to teonet
	for teo.Connect() != nil {
		time.Sleep(1 * time.Second)
	}

	// Teonet address
	fmt.Printf("Teonet addres: %s\n\n", teo.Address())

	// Connect to monitor
	teomon.Connect(teo, monitor, teomon.Metric{
		AppName:    appName,
		AppShort:   appShort,
		AppVersion: appVersion,
		TeoVersion: teonet.Version,
	})

	select {} // sleep forever
}
