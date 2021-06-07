// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet command api module

package teonet

import "fmt"

// APInterface is teonet api interface
type APInterface interface {
	Name() string
	Short() string
	Long() string
	Cmd() byte
	ExecMode() APIexecMode
	Reader(c *Channel, data []byte) bool
}

// APIexecMode how to execute command: Server - execute command if there is server
// connection; Client - execute command if ther is client connection; Both -
// execute command if there is any server or client connection
type APIexecMode byte

const (
	// Server - execute command if there is server connection
	Server APIexecMode = iota

	// Client - execute command if there is client connection
	Client

	// Both - execute command if there is any server or client connection
	Both
)

// AutoCmd sent command number automaticly (next number after previouse command)
// const AutoCmd = 0

// MakeAPI is teonet API interface builder
func MakeAPI(name, short, long string, cmd byte, execMode APIexecMode,
	reader func(c *Channel, data []byte) bool) APInterface {
	apiData := &makeAPIData{
		name:     name,
		shor:     short,
		long:     long,
		cmd:      cmd,
		reader:   reader,
		execMode: execMode,
	}
	return apiData
}

// makeAPIData is teonet API interface builder data
type makeAPIData struct {
	name     string
	shor     string
	long     string
	cmd      byte
	execMode APIexecMode
	reader   func(c *Channel, data []byte) bool
}

func (a makeAPIData) Name() string          { return a.name }
func (a makeAPIData) Short() string         { return a.shor }
func (a makeAPIData) Long() string          { return a.long }
func (a makeAPIData) Cmd() byte             { return a.cmd }
func (a makeAPIData) ExecMode() APIexecMode { return a.execMode }
func (a makeAPIData) Reader(c *Channel, data []byte) bool {
	return a.reader(c, data)
}

// NewAPI create new teonet api
func NewAPI(teo *Teonet) *API {
	return &API{Teonet: teo}
}

// API teonet api receiver
type API struct {
	*Teonet
	cmds []APInterface // APIData
}

// Add api command
func (a *API) Add(cmds ...APInterface) {
	a.cmds = append(a.cmds, cmds...)
}

// Reader api commands reader
func (a API) Reader() func(c *Channel, p *Packet, err error) (processed bool) {
	return func(c *Channel, p *Packet, err error) (processed bool) {
		// Skip packet with error
		if err != nil {
			return false
		}

		// Parse command
		cmd := a.Command(p.Data())

		// Select and Execute commands readers
		for i := range a.cmds {

			switch {
			// Check if we can execute this command depend of ExecMode
			case !a.canExecute(a.cmds[i], c):
				continue

			// Check command number
			case a.cmds[i].Cmd() != cmd.Cmd:
				continue

			// Execute command
			case a.cmds[i].Reader(c, cmd.Data):
				return true
			}

			// All done in 'unic command mode' when only one command with this
			// number may be added
			break
		}
		return
	}
}

// String return strin with api commands
func (a API) String() (str string) {
	// Calculate name lenngth
	var max int
	for i := range a.cmds {
		if l := len(a.cmds[i].Name()); l > max {
			max = l
		}
	}
	// Create output string
	for i := range a.cmds {
		if i > 0 {
			str += "\n"
		}
		str += fmt.Sprintf("%-*s %3d - %s", max+3, a.cmds[i].Name(), a.cmds[i].Cmd(), a.cmds[i].Short())
	}
	return
}

// canExecute check can execute this command
func (a API) canExecute(api APInterface, c *Channel) bool {
	switch api.ExecMode() {
	case Both:
		return true
	case Server:
		return c.ServerMode()
	case Client:
		return c.ClientMode()
	}
	return false
}
