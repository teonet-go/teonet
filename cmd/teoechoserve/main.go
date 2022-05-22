package main

import (
	"fmt"

	"github.com/kirill-scherba/teonet"
)

const (
	appName    = "Teonet echo server sample application"
	appShort   = "teoechoserve"
	appVersion = "0.0.1"
)

func main() {

	// Teonwt application logo
	teonet.Logo(appName, appVersion)

	// Start Teonet client
	teo, err := teonet.New(appShort,
		// Main application reader - receive and process incoming messages
		func(c *teonet.Channel, p *teonet.Packet, e *teonet.Event) bool {

			// Skip not Data events
			if e.Event != teonet.EventData {
				return false
			}

			// In server mode
			if c.ServerMode() {

				// Print received message
				fmt.Printf("got from %s, \"%s\", len: %d, id: %d, tt: %6.3fms\n",
					c, p.Data(), len(p.Data()), p.ID(),
					float64(c.Triptime().Microseconds())/1000.0,
				)

				// Send answer
				answer := []byte("Teonet answer to " + string(p.Data()))
				c.Send(answer)
			}

			return true
		},
	)
	if err != nil {
		panic("can't init Teonet, error: " + err.Error())
	}

	// Connect to Teonet
	err = teo.Connect()
	if err != nil {
		panic("can't connect to Teonet, error: " + err.Error())
	}

	// Print application address
	addr := teo.Address()
	fmt.Println("Connected to teonet, this app address:", addr)

	// Wait forever
	select {}
}
