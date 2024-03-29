// Teonet echo client/server sample application
package main

import (
	"flag"
	"os"
	"time"

	"github.com/teonet-go/teomon"
	"github.com/teonet-go/teonet"
)

const (
	appName    = "Teonet echo client/server sample application"
	appShort   = "teoecho"
	appVersion = teonet.Version
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
		teo.Log().Debug.Printf("got from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n",
			c, p.Data(), len(p.Data()), p.ID(),
			float64(c.Triptime().Microseconds())/1000.0,
		)

		// Send answer
		answer := []byte("Teonet answer to " + string(p.Data()))
		c.Send(answer)
	}

	return true
}

func main() {

	// Application logo
	teonet.Logo(appName, appVersion)

	// Parse applications flags
	var p struct {
		appShort    string
		port        int
		stat        bool
		hotkey      bool
		showPrivate bool
		sendTo      string
		sendDelay   int
		logLevel    string
		logFilter   string
		monitor     string
	}
	flag.StringVar(&p.appShort, "name", appShort, "application short name")
	flag.IntVar(&p.port, "p", 0, "local port")
	flag.BoolVar(&p.stat, "stat", false, "show trudp statistic")
	flag.BoolVar(&p.hotkey, "hotkey", false, "start hotkey menu")
	flag.BoolVar(&p.showPrivate, "show-private", false, "show private key")
	flag.StringVar(&p.sendTo, "send-to", "", "send messages to address")
	flag.IntVar(&p.sendDelay, "send-delay", 0, "delay between send message in milleseconds")
	flag.StringVar(&p.logLevel, "loglevel", "NONE", "log level")
	flag.StringVar(&p.logFilter, "logfilter", "", "log filter")
	flag.StringVar(&p.monitor, "monitor", "", "monitor address")
	flag.Parse()

	// Start teonet client
	teo, err := teonet.New(p.appShort, p.port, reader, teonet.Stat(p.stat),
		teonet.Hotkey(p.hotkey), p.logLevel, teonet.Logfilter(p.logFilter),
	)
	if err != nil {
		panic("can't init Teonet, error: " + err.Error())
	}

	// Show this application private key
	if p.showPrivate {
		teo.Log().Debug.Printf("%s\n", teo.GetPrivateKey())
		os.Exit(0)
	}

connect:
	// Connect to teonet
	err = teo.Connect()
	if err != nil {
		teo.Log().Debug.Println("can't connect to Teonet, error:", err)
		time.Sleep(1 * time.Second)
		goto connect
	}

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

	// Sleep forever if sendTo flag does not set (in server mode)
	if p.sendTo == "" {
		select {}
	}

connectto:
	// Connect to Peer (selected in send-to application flag) and receive
	// packets in own reader
	if err := teo.ConnectTo(p.sendTo,
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
			teo.Log().Debug.Printf("got(r) from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n\n",
				c, p.Data(), len(p.Data()), p.ID(), float64(c.Triptime().Microseconds())/1000.0,
			)
			processed = true

			return
		},
	); err != nil {
		teo.Log().Debug.Println("can't connect to Peer, error:", err)
		time.Sleep(1 * time.Second)
		goto connectto
	}

sendto:
	// Send to Peer
	if p.sendDelay > 0 {
		time.Sleep(time.Duration(p.sendDelay) * time.Millisecond)
	}
	teo.Log().Debug.Println("send message to", p.sendTo)
	if _, err := teo.SendTo(p.sendTo, []byte("Hello world!")); err != nil {
		teo.Log().Debug.Println(err)
	}
	goto sendto
}
