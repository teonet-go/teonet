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
	appVersion = "0.3.0"

	// apis = "WXJfYLDEtg6Rkm1OHm9I9ud9rR6qPlMH6NE"
	apis = "LYfwf3tivLoJ5xH2GM4MeJu1GgiezdBj7Er"
)

func main() {

	// Application logo
	teonet.Logo(appName, appVersion)

	// Parse applications flags
	var p struct {
		appShort  string
		port      int
		stat      bool
		hotkey    bool
		logLevel  string
		logFilter string
		connectTo string
	}
	flag.StringVar(&p.appShort, "name", appShort, "application short name")
	flag.IntVar(&p.port, "p", 0, "local port")
	flag.BoolVar(&p.stat, "stat", false, "show trudp statistic")
	flag.BoolVar(&p.hotkey, "hotkey", false, "start hotkey menu")
	flag.StringVar(&p.connectTo, "connect-to", apis, "connect to api server")
	flag.StringVar(&p.logLevel, "loglevel", "NONE", "log level")
	flag.StringVar(&p.logFilter, "logfilter", "", "log filter")
	flag.Parse()

	if p.connectTo == "" {
		fmt.Println("Flag -log-level should be set")
		flag.Usage()
		return
	}

	// Start teonet (client or server)
	teo, err := teonet.New(p.appShort, p.port, teonet.Stat(p.stat),
		teonet.Hotkey(p.hotkey), p.logLevel, teonet.Logfilter(p.logFilter))
	if err != nil {
		panic("can't init Teonet, error: " + err.Error())
	}

	teo.Log().Debug.Println("Start")

	// Connect to teonet
	for teo.Connect("http://localhost:10000/auth") != nil {
		// teo.Log().Debug.Println("can't connect to Teonet, error:", err)
		time.Sleep(1 * time.Second)
	}

	// Teonet address
	fmt.Printf("Teonet addres: %s\n\n", teo.Address())

	// Connect to API server (selected in connect-to application flag) and receive
	// packets in own reader. Use WXJfYLDEtg6Rkm1OHm9I9ud9rR6qPlMH6NE addres to
	// connect to installed teoapi example.
	var stopChannelReader bool
	for {
		err := teo.ConnectTo(p.connectTo,
			// Receive and process packets from this channel(address). Return
			// true if packet processed. If return false package will processed
			// by other readers include main application reader (just comment
			// 'processed = true' line and you'll see two 'got from ...' message)
			func(c *teonet.Channel, p *teonet.Packet, e *teonet.Event) (processed bool) {
				if e.Event == teonet.EventData && !stopChannelReader {
					// Print received message
					// teo.Log().Debug.Printf("got(r) from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n\n",
					// 	c, p.Data(), len(p.Data()), p.ID(), float64(c.Triptime().Microseconds())/1000.0,
					// )
					teo.Log().Debug.Printf("Got '%s', from %s\n", p.Data(), c)
					processed = true
				}
				return
			},
		)
		if err == nil {
			break
		}
		teo.Log().Debug.Println("can't connect to API server, error:", err)
		time.Sleep(1 * time.Second)
	}

	// Connect message
	teo.Log().Debug.Printf("Connected to API sample server: %s\n\n", p.connectTo)

	// Test #1: Send Teonet Commands -------------------------------------------
	teo.Log().Debug.Printf("===> Test #1: Send commands to API server with Teonet Command Send and Get answer in Connect Reader\n\n")

	// Send command 129('hello')
	data := []byte("Kirill")
	teo.Log().Debug.Println("Send 129('hello') with data:", string(data))
	teo.Command(129, []byte("Kirill")).SendTo(p.connectTo)

	// Send command 130('description')
	teo.Log().Debug.Println("Send 130('description') without data")
	teo.Command(130, nil).SendTo(p.connectTo)

	time.Sleep(250 * time.Millisecond)
	stopChannelReader = true

	// Test #2: Create Teonet client API interface -----------------------------
	apicli, _ := teo.NewAPIClient(p.connectTo)
	teo.Log().Debug.Printf("\n\n===> Test #2: Create API interface and Get servers APIData, Name: %s\n\n", apicli.Apis[0].Name())

	// Test #3: Send commands by number ----------------------------------------
	teo.Log().Debug.Printf("===> Test #3: Send commands by number\n\n")

	// Send command #129 and wait answer
	cmd := byte(129)
	teo.Log().Debug.Printf("Send cmd=%d 'Kirill'\n", cmd)
	apicli.SendTo(cmd, []byte("Kirill"))
	data, err = apicli.WaitFrom(cmd)
	if err != nil {
		teo.Log().Debug.Printf("can't got cmd=%d data, err: %s\n", cmd, err)
	}
	teo.Log().Debug.Printf("Got  cmd=%d '%s'\n\n", cmd, data)

	// Send command #130 and wait answer
	cmd = 130
	teo.Log().Debug.Printf("Send cmd=%d \n", cmd)
	apicli.SendTo(cmd, []byte("Kirill"))
	data, err = apicli.WaitFrom(cmd)
	if err != nil {
		teo.Log().Debug.Printf("can't got cmd=%d data, err: %s\n", cmd, err)
	}
	teo.Log().Debug.Printf("Got  cmd=%d '%s'\n\n", cmd, data)

	// Test #4: Send commands by name ------------------------------------------
	teo.Log().Debug.Printf("===> Test #4: Send commands by name\n\n")

	// Send command 'hello' and wait answer
	cmdName := "hello"
	teo.Log().Debug.Printf("Send cmd='%s' 'Kirill'\n", cmdName)
	apicli.SendTo(cmdName, []byte("Kirill"))
	data, err = apicli.WaitFrom(cmdName)
	if err != nil {
		teo.Log().Debug.Printf("can't got cmd='%s' data, err: %s\n", cmdName, err)
	}
	teo.Log().Debug.Printf("Got  cmd='%s' '%s'\n\n", cmdName, data)

	// Send command 'description' and wait answer
	cmdName = "description"
	teo.Log().Debug.Printf("Send cmd='%s'\n", cmdName)
	apicli.SendTo(cmdName, nil)
	data, err = apicli.WaitFrom(cmdName)
	if err != nil {
		teo.Log().Debug.Printf("can't got cmd='%s' data, err: %s\n", cmdName, err)
	}
	teo.Log().Debug.Printf("Got  cmd='%s' '%s'\n\n", cmdName, data)

	// Test #5: Send and wait in send function ---------------------------------
	teo.Log().Debug.Printf("===> Test #5: Send and wait in send function\n\n")

	cmdName1 := "hello"
	teo.Log().Debug.Printf("Send cmd='%s' 'Kirill'\n", cmdName1)
	apicli.SendTo(cmdName1, []byte("Kirill"), func(data []byte, err error) {
		if err != nil {
			teo.Log().Debug.Printf("can't got cmd='%s' data, err: %s\n", cmdName1, err)
			return
		}
		teo.Log().Debug.Printf("Got  cmd='%s' '%s'\n", cmdName1, data)
	})

	cmdName2 := "description"
	teo.Log().Debug.Printf("Send cmd='%s'\n", cmdName2)
	apicli.SendTo(cmdName2, nil, func(data []byte, err error) {
		if err != nil {
			teo.Log().Debug.Printf("can't got cmd='%s' data, err: %s\n", cmdName2, err)
			return
		}
		teo.Log().Debug.Printf("Got  cmd='%s' '%s'\n\n", cmdName2, data)
	})

	time.Sleep(500 * time.Millisecond)
	teo.Close()
	teo.Log().Debug.Println("All done, quit...")

	// select {} // sleep forever
}
