package main

import (
	"fmt"
	"time"

	"github.com/kirill-scherba/teonet"
)

const (
	appName    = "Teonet echo client sample application"
	appShort   = "teoechocli"
	appVersion = "0.0.1"
	echoServer = "OUPdQ35M0x53ScObAMWiLaDv1Kn6q7KdO61" // "dBTgSEHoZ3XXsOqjSkOTINMARqGxHaXIDxl"
	sendDelay  = 3000
)

func main() {

	// Teonwt application logo
	teonet.Logo(appName, appVersion)

	// Start Teonet client
	teo, err := teonet.New(appShort)
	if err != nil {
		panic("can't init Teonet, error: " + err.Error())
	}

	// Connect to Teonet
	err = teo.Connect()
	if err != nil {
		teo.Log().Debug.Println("can't connect to Teonet, error:", err)
		panic("can't connect to Teonet, error: " + err.Error())
	}

	// Connect to echo server
	err = teo.ConnectTo(echoServer,

		// Get messages from echo server
		func(c *teonet.Channel, p *teonet.Packet, e *teonet.Event) (proc bool) {

			// Skip not Data Events
			if e.Event != teonet.EventData {
				return
			}

			// Print received message
			fmt.Printf("got from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n\n",
				c, p.Data(), len(p.Data()), p.ID(),
				float64(c.Triptime().Microseconds())/1000.0,
			)
			proc = true

			return
		},
	)

	// Send messages to echo server
	for {
		data := []byte("Hello world!")
		fmt.Printf("send to  %s, \"%s\", len: %d\n", echoServer, data, len(data))
		_, err = teo.SendTo(echoServer, []byte("Hello world!"))
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(time.Duration(sendDelay) * time.Millisecond)
	}
}
