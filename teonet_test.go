package teonet

import (
	"fmt"
	"log"
	"sync"
	"testing"
)

func TestSendPackagesInMoment(t *testing.T) {

	const numToSend = 10
	var wg sync.WaitGroup

	log.Println("start connecting")

	// Teonet client 1
	teocli1, err := New("teocli1", func(c *Channel, p *Packet, e *Event) bool {
		if e.Event != EventData {
			return false
		}
		log.Printf("teocli1 got from %s data: %s\n", c.Address(), string(p.Data()))
		wg.Done()
		return true
	})
	if err != nil {
		t.Error("can't create teonet client:", err)
		return
	}
	err = teocli1.Connect()
	if err != nil {
		t.Error("can't connect to teonet:", err)
		return
	}

	// Teonet client 2
	teocli2, err := New("teocli2", func(c *Channel, p *Packet, e *Event) bool {
		if e.Event != EventData {
			return false
		}
		log.Printf("teocli2 got from %s data: %s\n", c.Address(), string(p.Data()))
		c.Send([]byte("Answer to " + string(p.Data())))
		return true
	})
	if err != nil {
		t.Error("can't create teonet client:", err)
		return
	}
	err = teocli2.Connect()
	if err != nil {
		t.Error("can't connect to teonet:", err)
		return
	}
	addr2 := teocli2.Address()

	// Connect Teonet-client-1 to Teonet-client-2
	teocli1.ConnectTo(addr2)
	if err != nil {
		t.Error("can't connect to teocli2:", err)
		return
	}

	// Teonet Client1 Send 'numToSend' packages from teoclient1 to teoclient2
	// go func() {
	for i := 0; i < numToSend; i++ {
		teocli1.SendTo(addr2, []byte(fmt.Sprintf("Hello %d from Client1", i+1)))
		wg.Add(1)
	}
	// }()

	wg.Wait()
	// select {}
}
