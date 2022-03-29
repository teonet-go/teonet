// Copyright 2021-22 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet command api module

package teonet

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/kirill-scherba/bslice"
)

// ApiInterface
type ApiInterface interface {
	ProcessPacket(p interface{})
}

// addApiReader sets teonet reader. This reader process received API commands
func (teo *Teonet) addApiReader(api ApiInterface) {
	if api == nil {
		return
	}
	teo.clientReaders.add(func(teo *Teonet, c *Channel, p *Packet, e *Event) (ret bool) {
		// Process API commands
		if e.Event == EventData {
			api.ProcessPacket(p.setCommandMode())
		}
		return
	})
}

// APInterface is teonet api interface
type APInterface interface {
	Name() string
	Short() string
	Long() string
	Usage() string
	Ret() string
	Cmd() byte
	ExecMode() (APIconnectMode, APIanswerMode)
	Reader(c *Channel, p *Packet, data []byte) bool
	Reader2(data []byte, answer func(data []byte)) bool
}

// API teonet api receiver
type API struct {
	*Teonet
	name    string        // API (application) name
	short   string        // API short name
	long    string        // API decription (or long name)
	version string        // API version
	cmds    []APInterface // API commands
	cmd     byte          // API cmdApi command number
	bslice.ByteSlice
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
	reader func(c *Channel, p *Packet, data []byte) bool,
	reader2 func(data []byte, answer func(data []byte)) bool,
) APInterface {
	apiData := &APIData{
		name:        name,
		short:       short,
		long:        long,
		usage:       usage,
		ret:         ret,
		cmd:         cmd,
		reader:      reader,
		reader2:     reader2,
		connectMode: execMode,
		answerMode:  answerMode,
	}
	return apiData
}

// MakeAPI2 is second teonet API interface builder
func MakeAPI2() *APIData {
	return &APIData{
		connectMode: ServerMode,
		answerMode:  CmdAnswer,
		reader: func(c *Channel, p *Packet, data []byte) bool {
			return true
		},
		reader2: func(data []byte, answer func(data []byte)) bool {
			return true
		},
	}
}

// NewAPI create new teonet api
func (teo *Teonet) NewAPI(name, short, long, version string, cmdAPIs ...byte) (api *API) {
	api = &API{
		Teonet:  teo,
		name:    name,
		short:   short,
		long:    long,
		version: version,
	}
	var cmdApi APInterface
	var cmd byte = CmdServerAPI
	if len(cmdAPIs) > 0 {
		cmd = cmdAPIs[0]
	}
	cmdApi = MakeAPI2().SetName("api").SetCmd(cmd).SetShort("get api").SetReturn("<api APIDataAr>").
		SetConnectMode(AnyMode).SetAnswerMode(CmdAnswer).
		SetReader(func(c *Channel, p *Packet, data []byte) bool {
			_, answerMode := cmdApi.ExecMode()
			log.Debug.Println("got api request, cmd:", cmdApi.Cmd(), p.From(), cmd,"answerMode:", answerMode)
			outData, _ := api.MarshalBinary()
			api.SendAnswer(cmdApi, c, outData, p)
			return true
		})
	api.Add(cmdApi)
	return api
}

// Short get app short name
func (a API) Short() string {
	return a.short
}

// Send answer to request
func (a *API) SendAnswer(cmd APInterface, c *Channel, data []byte, p *Packet) (id uint32, err error) {

	// Get answer mode
	_, answerMode := cmd.ExecMode()
	if answerMode&PacketIDAnswer > 0 {
		id := make([]byte, 4)
		binary.LittleEndian.PutUint32(id, uint32(p.ID()))
		data = append(id, data...)
	}

	// Send answer
	if answerMode&CmdAnswer > 0 {
		a.Command(cmd.Cmd(), data).Send(c)
	} else {
		c.Send(data)
	}

	return
}

// Send answer to request
func (a *API) SendAnswer2(data []byte, answer func(data []byte)) (id uint32, err error) {
	answer(data)
	return
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

// Reader process teonet commands as described in API
func (a API) Reader() func(c *Channel, p *Packet, e *Event) (processed bool) {
	return func(c *Channel, p *Packet, e *Event) (processed bool) {
		// Skip not Data Events
		if e.Event != EventData {
			return
		}
		// Execute reader
		return a.readerExec(
			p.Data(),
			func(i int) bool { return a.canExecute(a.cmds[i], c) },
			func(i int, data []byte) bool { return a.cmds[i].Reader(c, p, data) },
		)
	}
}

// Reader2 process not teonet (webrtc for example) commands as described in API
func (a API) Reader2() func(data []byte, answer func(data []byte)) (processed bool) {
	return func(data []byte, answer func(data []byte)) (processed bool) {
		return a.readerExec(
			data,
			func(i int) bool { return true },
			func(i int, data []byte) bool { return a.cmds[i].Reader2(data, answer) },
		)
	}
}

// readerExec parce and execute command
func (a API) readerExec(data []byte, canExecute func(i int) bool,
	execute func(i int, data []byte) bool) (processed bool) {

	// Parse command
	cmd := a.Command(data)

	// Select and Execute commands readers
	for i := range a.cmds {

		switch {
		// Check if we can execute this command depend of ExecMode
		case !canExecute(i):
			continue

		// Check command number
		case a.cmds[i].Cmd() != cmd.Cmd:
			continue

		// Execute command
		case execute(i, cmd.Data):
			return true
		}

		// All done in 'unic command mode' when only one command with this
		// number may be added
		break
	}

	return
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

// String is API stringlify, it return help text in string
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

// makeAPIData make APIData struct
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

// MarshalBinary binary marshal API
func (a API) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	a.WriteSlice(buf, []byte(a.name))
	a.WriteSlice(buf, []byte(a.short))
	a.WriteSlice(buf, []byte(a.long))
	a.WriteSlice(buf, []byte(a.version))
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
	name    string    // API (application) name
	short   string    // API short name
	long    string    // API decription (or long name)
	version string    // API version
	Apis    []APIData // API commands data
	bslice.ByteSlice
}

// UnmarshalBinary binary unmarshal APIDataAr
func (a *APIDataAr) UnmarshalBinary(data []byte) (err error) {
	var buf = bytes.NewBuffer(data)

	if a.name, err = a.ReadString(buf); err != nil {
		return
	}
	if a.short, err = a.ReadString(buf); err != nil {
		return
	}
	if a.long, err = a.ReadString(buf); err != nil {
		return
	}
	if a.version, err = a.ReadString(buf); err != nil {
		return
	}
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
