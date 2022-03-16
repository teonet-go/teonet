// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet event module

package teonet

// Teonet event struct
type Event struct {
	Event TeonetEventType
	Err   error
}

// Teonet event type
type TeonetEventType byte

// Teonet events
const (
	EventNone TeonetEventType = iota

	// Event when Teonet client initialized and start listen, Err = nil
	EventTeonetInit

	// Event when Connect to teonet r-host, Err = nil
	EventTeonetConnected

	// Event when Disconnect from teonet r-host, Err = dosconnect error
	EventTeonetDisconnected

	// Event when Connect to peer, Err = nil
	EventConnected

	// Event when Disconnect from peer, Err = dosconnect error
	EventDisconnected

	// Event when Data Received, Err = nil
	EventData
)

// Event to string
func (e Event) String() (str string) {
	switch e.Event {
	case EventNone:
		str = "EventNone"
	case EventTeonetInit:
		str = "EventTeonetInit"
	case EventTeonetConnected:
		str = "EventTeonetConnected"
	case EventTeonetDisconnected:
		str = "EventTeonetDisconnected"
	case EventConnected:
		str = "EventConnected"
	case EventDisconnected:
		str = "EventDisconnected"
	case EventData:
		str = "EventData"
	}
	return
}
