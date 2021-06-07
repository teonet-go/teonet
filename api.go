// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet command api module

package teonet

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// APInterface is teonet api interface
type APInterface interface {
	Name() string
	Short() string
	Long() string
	Usage() string
	Ret() string
	Cmd() byte
	ExecMode() (APIconnectMode, APIanswerMode)
	Reader(c *Channel, data []byte) bool
}

// APIconnectMode connection type of received command:
//
//   Server: execute command if there is server connection;
//   Client: execute command if ther is client connection;
//   Both: execute command if there is any server or client connection
//
// Server connection mode: any Peer connected to this application with function
// ConnectTo (and Peer send commands to this application). Client connection mode:
// this application connected to Peer with function ConnectTo (and Peer send
// commands to this application)
type APIconnectMode byte

const (
	// ServerMode - execute command if there is server connection
	ServerMode APIconnectMode = 1 << iota

	// ClientMode - execute command if there is client connection
	ClientMode

	// AnyMode - execute command if there is any server or client connection
	AnyMode = ClientMode & ServerMode
)

// APIexecMode how to answer to this command will be send. Constan may be
// combined, f.e. answer with Command and ID and Data:
// answerMode = CmdAnswer | PacketIDAnswer | DataAnswer
type APIanswerMode byte

const (
	// DataAnswer - send data in answer
	DataAnswer APIanswerMode = 1 << iota

	// CmdAnswer - send command in answer
	CmdAnswer

	// PacketIDAnswer - send received packet ID in answer
	PacketIDAnswer

	// NoAnswer - answer does not send
	NoAnswer APIanswerMode = 0
)

// MakeAPI is teonet API interface builder
func MakeAPI(name, short, long, usage, ret string, cmd byte,
	execMode APIconnectMode, answerMode APIanswerMode,
	reader func(c *Channel, data []byte) bool) APInterface {
	apiData := &APIData{
		name:        name,
		short:       short,
		long:        long,
		usage:       usage,
		ret:         ret,
		cmd:         cmd,
		reader:      reader,
		connectMode: execMode,
		answerMode:  answerMode,
	}
	return apiData
}

// APIData is teonet API interface builder data
type APIData struct {
	name        string
	short       string
	long        string
	usage       string
	ret         string
	cmd         byte
	connectMode APIconnectMode
	answerMode  APIanswerMode
	reader      func(c *Channel, data []byte) bool
	ByteSlice
}

func (a APIData) Name() string  { return a.name }
func (a APIData) Short() string { return a.short }
func (a APIData) Long() string  { return a.long }
func (a APIData) Usage() string { return a.usage }
func (a APIData) Ret() string   { return a.ret }
func (a APIData) Cmd() byte     { return a.cmd }
func (a APIData) ExecMode() (APIconnectMode, APIanswerMode) {
	return a.connectMode, a.answerMode
}
func (a APIData) Reader(c *Channel, data []byte) bool {
	return a.reader(c, data)
}

// NewAPI create new teonet api
func NewAPI(teo *Teonet) *API {
	return &API{Teonet: teo}
}

// API teonet api receiver
type API struct {
	*Teonet
	// appName        string
	// appVersion     string
	// appDescription string
	cmds []APInterface
	cmd  byte
}

// Cmd return API command number and save this command to use in CmdNext
func (a *API) Cmd(cmd byte) byte {
	a.cmd = cmd
	return cmd
}

// CmdNext return next API command number
func (a *API) CmdNext() byte {
	a.cmd++
	return a.cmd
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
func (a API) Help(shorts ...bool) (str string) {
	var short bool
	if len(shorts) > 0 {
		short = shorts[0]
	}
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
			if !short {
				str += "\n"
			}
		}
		if short {
			str += fmt.Sprintf("%-*s %3d - %s", max, a.cmds[i].Name(), a.cmds[i].Cmd(), a.cmds[i].Short())
			continue
		}
		str += fmt.Sprintf("%-*s %s\n", max, a.cmds[i].Name(), a.cmds[i].Short())
		str += fmt.Sprintf("%*s command: %d\n", max, "", a.cmds[i].Cmd())
		str += fmt.Sprintf("%*s usage:   %s\n", max, "", a.cmds[i].Name()+" "+a.cmds[i].Usage())
		str += fmt.Sprintf("%*s return:  %s", max, "", a.cmds[i].Ret())
	}
	return
}

func (a API) String() (str string) {
	return a.Help()
}

// canExecute check can execute this command
func (a API) canExecute(api APInterface, c *Channel) bool {
	connectMode, _ := api.ExecMode()
	switch connectMode {
	case AnyMode:
		return true
	case ServerMode:
		return c.ServerMode()
	case ClientMode:
		return c.ClientMode()
	}
	return false
}

func (a API) makeAPIData(in APInterface) (ret *APIData) {
	connectMode, answerMode := in.ExecMode()
	ret = &APIData{
		name:        in.Name(),
		short:       in.Short(),
		long:        in.Long(),
		usage:       in.Usage(),
		ret:         in.Ret(),
		cmd:         in.Cmd(),
		connectMode: connectMode,
		answerMode:  answerMode,
	}
	return
}

func (a APIData) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	a.WriteSlice(buf, []byte(a.name))
	a.WriteSlice(buf, []byte(a.short))
	a.WriteSlice(buf, []byte(a.long))
	a.WriteSlice(buf, []byte(a.usage))
	a.WriteSlice(buf, []byte(a.ret))
	binary.Write(buf, binary.LittleEndian, a.cmd)
	binary.Write(buf, binary.LittleEndian, a.connectMode)
	binary.Write(buf, binary.LittleEndian, a.answerMode)

	data = buf.Bytes()
	return
}

func (a *APIData) UnmarshalBinary(buf *bytes.Buffer /*data []byte*/) (err error) {
	// var buf = bytes.NewBuffer(data)

	if a.name, err = a.ReadString(buf); err != nil {
		return
	}
	if a.short, err = a.ReadString(buf); err != nil {
		return
	}
	if a.long, err = a.ReadString(buf); err != nil {
		return
	}
	if a.usage, err = a.ReadString(buf); err != nil {
		return
	}
	if a.ret, err = a.ReadString(buf); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &a.cmd); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &a.connectMode); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &a.answerMode); err != nil {
		return
	}

	return
}

func (a API) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	numCmds := uint16(len(a.cmds))
	binary.Write(buf, binary.LittleEndian, numCmds)
	for i := range a.cmds {
		data, _ := a.makeAPIData(a.cmds[i]).MarshalBinary()
		binary.Write(buf, binary.LittleEndian, data)
	}
	data = buf.Bytes()
	return
}

type APIDataAr struct {
	Apis []APIData
	ByteSlice
}

func (a *APIDataAr) UnmarshalBinary(data []byte) (err error) {
	var buf = bytes.NewBuffer(data)

	var numCmds uint16
	if err = binary.Read(buf, binary.LittleEndian, &numCmds); err != nil {
		return
	}
	for i := 0; i < int(numCmds); i++ {
		var api APIData
		if err = api.UnmarshalBinary(buf); err != nil {
			return
		}
		a.Apis = append(a.Apis, api)
	}

	return
}
