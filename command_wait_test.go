package teonet

import (
	"fmt"
	"testing"
	"time"

	"github.com/kirill-scherba/tru/teolog"
)

func TestWaitAnswer(t *testing.T) {

	log := teolog.New()
	log.SetLevel(teolog.Debug)

	// Init teonet
	teo, err := New("TestWaitAnswer")
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("MakeWaitAttr", func(t *testing.T) {
		attr := teo.MakeWaitAttr().Cmd(129).ID(11).Func(func([]byte) bool { return true }).Timeout(5 * time.Second)
		fmt.Println(attr)
		if err != nil {
			t.Error(err)
		}
	})

	// Connect to teonet
	err = teo.Connect()
	if err != nil {
		t.Error(err)
	}

	// Peer alias
	const (

		// Echo server
		echo = "dBTgSEHoZ3XXsOqjSkOTINMARqGxHaXIDxl"

		// API server
		apis = "WXJfYLDEtg6Rkm1OHm9I9ud9rR6qPlMH6NE"
	)

	// Connect to echo server
	err = teo.ConnectTo(echo)
	if err != nil {
		t.Error(err)
	}

	t.Run("WaitFromData", func(t *testing.T) {

		msg := "Hello!"
		log.Debug.Println("send data:", msg)
		_, err = teo.SendTo(echo, []byte(msg))
		if err != nil {
			t.Error(err)
		}

		data, err := teo.WaitFrom(echo)
		if err != nil {
			t.Error(err)
		}
		log.Debug.Println("got answer:", string(data))

	})

	// Connect to api server
	err = teo.ConnectTo(apis)
	if err != nil {
		t.Error(err)
	}

	// Send command to peer and wait answer with WaitFrom
	t.Run("WaitFromCmd", func(t *testing.T) {

		cmd := 129
		name := "Kirill!"
		log.Debug.Println("send cmd", cmd, "data:", name)

		_, err = teo.Command(cmd, []byte(name)).SendTo(apis)
		if err != nil {
			t.Error(err)
			return
		}

		// This command return data without command, if cmd returned should add
		// cmd to attr WaitFrom parameter
		data, err := teo.WaitFrom(apis)
		if err != nil {
			t.Error(err)
			return
		}

		log.Debug.Println("got answer:", string(data))
	})

	// Create reader with MakeReader. Created reader understand command, id, data
	// func attributes. In this test we send command to peer and got answer
	// inside reader added to SendTo
	t.Run("MakeReader", func(t *testing.T) {

		cmd := 129
		name := "Kirill!"
		log.Debug.Println("send cmd", cmd, "data:", name)

		wait := make(chan interface{})

		if _, err = teo.Command(cmd, []byte(name)).
			SendTo(apis, teo.MakeWaitReader(func(data []byte) bool {
				log.Debug.Println("got answer:", string(data))
				wait <- struct{}{}
				return true
			}).Reader()); err != nil {

			t.Error(err)
		}

		<-wait
	})
}
