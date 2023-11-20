// Copyright 2021-2023 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet api client module

package teonet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/kirill-scherba/bslice"
)

const (
	// Get server api command
	CmdServerAPI = 255
	// Get server api command
	CmdClientAPI = 254
)

const (
	FmtMsgCommandNotCount = "command '%s' not found"
)

var (
	ErrWoronCommand = errors.New("wrong command")
)

// APIClient contains clients api data and receive methods
type APIClient struct {
	APIDataAr
	address string
	cmdAPI  byte
	teo     *Teonet
}
type APIDataAr struct {
	name      string      // API (application) name
	short     string      // API short name
	long      string      // API decription (or long name)
	version   string      // API version
	Apis      []APIData   // API commands data
	UserField interface{} // Some user field
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

// NewAPIClient create new APIClient object
func (teo *Teonet) NewAPIClient(address string, cmdAPIs ...byte) (apicli *APIClient, err error) {
	apicli = new(APIClient)
	apicli.teo = teo
	apicli.address = address
	if len(cmdAPIs) > 0 {
		apicli.cmdAPI = cmdAPIs[0]
	} else {
		apicli.cmdAPI = CmdServerAPI
	}
	err = apicli.getApi()
	return
}

// SendTo sends api command.
func (api *APIClient) SendTo(command interface{}, data []byte,
	waits ...func(data []byte, err error)) (id int, err error) {

	cmd, err := api.GetCmd(command)
	if err != nil {
		return
	}
	id, err = api.teo.Command(cmd, data).SendTo(api.address)
	if len(waits) > 0 {
		go func() { waits[0](api.WaitFrom(cmd, uint32(id))) }()
	}
	return
}

// WaitFrom wait receiving data from peer. The third function parameter is
// timeout. It may be omitted or contain timeout time of time. Duration type.
// If timeout parameter is omitted than default timeout value sets to 2 second.
// Next parameter is checkDataFunc func([]byte) bool. This function calls to
// check packet data and returns true if packet data valid. This parameter may
// be omitted too.
func (api *APIClient) WaitFrom(command interface{}, packetID ...interface{}) (data []byte, err error) {

	// Get command number
	cmd, err := api.GetCmd(command)
	if err != nil {
		return
	}

	// Get answer mode
	var answerMode APIanswerMode
	// When we execute cmdAPI=255 the APIcommands is not loaded yet. The cmdAPI
	// always return: <cmdAPI byte><api APIDataAr>
	// So check cmdAPI first, than get answer mode
	if cmd == api.cmdAPI {
		answerMode = CmdAnswer
	} else {
		a, ok := api.AnswerMode(cmd)
		if !ok {
			err = ErrWoronCommand
			return
		}
		answerMode = a
	}

	// Set WaitFrom attributes depend of answer mode
	var attr []interface{}
	if answerMode&CmdAnswer > 0 {
		attr = append(attr, cmd)
	}
	if answerMode&PacketIDAnswer > 0 {
		attr = append(attr, packetID...)
	}

	// Wait result
	data, err = api.teo.WaitFrom(api.address, attr...)
	return
}

// Cmd get command number by name.
func (api *APIClient) Cmd(name string) (cmd byte, ok bool) {
	for i := range api.Apis {
		if api.Apis[i].name == name {
			cmd = api.Apis[i].cmd
			ok = true
			return
		}
	}
	return
}

// Return get return parameter by cmd number or name.
func (api *APIClient) Return(command interface{}) (ret string, ok bool) {
	a, ok := api.apiData(command)
	if ok {
		ret = a.ret
	}
	return
}

// AnswerMode get answer mode parameter by cmd number or name.
func (api *APIClient) AnswerMode(command interface{}) (ret APIanswerMode, ok bool) {
	a, ok := api.apiData(command)
	if ok {
		ret = a.answerMode
	}
	return
}

// String stringlify APIClient, return same string as Help function.
func (api APIClient) String() (str string) {
	str += api.Help(false)
	return
}

// APIClient return APICient help in string.
func (api APIClient) Help(short bool) (str string) {

	// Name version and description
	str += fmt.Sprintf("%s, ver %s\n", api.name, api.version)
	str += fmt.Sprintf("(short name: %s)\n\n", api.short)
	if api.long != "" {
		str += api.long + "\n\n"
	}

	// Calculate name lenngth
	var max int
	for i := range api.Apis {
		if l := len(api.Apis[i].Name()); l > max {
			max = l
		}
	}
	max += 2
	// Commands
	// TODO: make common function to get commands here and in api server print
	str += "API commands:\n\n"
	for i, a := range api.Apis {
		if i > 0 {
			str += "\n"
		}
		if short {
			str += fmt.Sprintf("%-*s %3d - %s", max, a.Name(), a.Cmd(), a.Short())
			continue
		}

		str += fmt.Sprintf("%-*s %s\n", max, a.Name(), a.Short())
		str += fmt.Sprintf("%*s cmd:    %d\n", max, "", a.Cmd())
		str += fmt.Sprintf("%*s usage:  %s\n", max, "", a.Name()+" "+a.Usage())
		var answer string
		if a.answerMode&CmdAnswer > 0 {
			answer += "<cmd byte>"
		}
		if a.answerMode&PacketIDAnswer > 0 {
			answer += "<packet_id uint32>"
		}
		answer += a.Ret()
		if answer != "" {
			str += fmt.Sprintf("%*s return: %s\n", max, "", answer)
		}
	}
	return
}

// Address returns application address.
func (api APIClient) Address() string { return api.address }

// AppShort returns application short name.
func (api APIClient) AppShort() string { return api.short }

// AppName returns application name.
func (api APIClient) AppName() string { return api.name }

// AppLong returns application long name (description).
func (api APIClient) AppLong() string { return api.long }

// GetCmd check command type and return command number.
func (api *APIClient) GetCmd(command interface{}) (cmd byte, err error) {
	switch v := command.(type) {
	case byte:
		cmd = v
	case int:
		cmd = byte(v)
	case string:
		var ok bool
		cmd, ok = api.Cmd(v)
		if !ok {
			err = fmt.Errorf(FmtMsgCommandNotCount, v)
			return
		}
	default:
		panic("wrong type of 'command' argument")
	}
	return
}

// DataserverIp gets dataserver ip address
func (api *APIClient) DataserverIp() (ip string) {
	ch, ok := api.teo.channels.get(api.address)
	if ok {
		ip = ch.Channel().Addr().(*net.UDPAddr).IP.String()
	}
	return
}

// apiData get return pointer to APIData by cmd number or name.
func (api *APIClient) apiData(command interface{}) (ret *APIData, ok bool) {
	cmd, err := api.GetCmd(command)
	if err != nil {
		return
	}
	for i := range api.Apis {
		if api.Apis[i].cmd == cmd {
			ret = &api.Apis[i]
			ok = true
			return
		}
	}
	return
}

// getApi send cmdAPI command and get answer with APIDataAr: all API definition.
func (api *APIClient) getApi() (err error) {
	api.SendTo(api.cmdAPI, nil)
	data, err := api.WaitFrom(api.cmdAPI)
	if err != nil {
		log.Error.Println("can't get api data, err", err)
		return
	}

	err = api.APIDataAr.UnmarshalBinary(data)
	if err != nil {
		log.Error.Println("can't unmarshal api data, err", err)
		return
	}

	return
}
