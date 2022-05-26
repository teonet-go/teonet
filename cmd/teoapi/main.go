// Teonet server with teonet api sample application
package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/teonet-go/teomon"
	"github.com/teonet-go/teonet"
)

const (
	appName    = "Teonet api server sample application"
	appShort   = "teoapi"
	appVersion = teonet.Version
	appLong    = ""
)

var appStartTime = time.Now()

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
	var p struct {
		appShort  string
		loglevel  string
		logfilter string
		monitor   string
		connectTo string
		hotkey    bool
		stat      bool
		port      int
	}
	flag.StringVar(&p.appShort, "name", appShort, "application short name")
	flag.IntVar(&p.port, "p", 0, "local port")
	flag.BoolVar(&p.stat, "stat", false, "show trudp statistic")
	flag.BoolVar(&p.hotkey, "hotkey", false, "start hotkey menu")
	flag.StringVar(&p.connectTo, "connect-to", "", "connect to api server")
	flag.StringVar(&p.loglevel, "loglevel", "NONE", "log level")
	flag.StringVar(&p.logfilter, "logfilter", "", "log filter")
	flag.StringVar(&p.monitor, "monitor", "", "monitor address")
	flag.Parse()

	// Start teonet (client or server)
	teo, err := teonet.New(p.appShort, p.port, teonet.Stat(p.stat),
		teonet.Hotkey(p.hotkey), p.loglevel, teonet.Logfilter(p.logfilter))
	if err != nil {
		panic("can't init Teonet, error: " + err.Error())
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
	fmt.Printf("Teonet address: %s\n\n", teo.Address())

	// Connect to monitor
	if len(p.monitor) > 0 {
		teomon.Connect(teo, p.monitor, teomon.Metric{
			AppName:      appName,
			AppShort:     appShort,
			AppVersion:   appVersion,
			TeoVersion:   teonet.Version,
			AppStartTime: appStartTime,
		})
	}

	select {} // sleep forever
}
