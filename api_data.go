// Copyright 2021-22 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet api data module

package teonet

import (
	"bytes"
	"encoding/binary"

	"github.com/kirill-scherba/bslice"
)

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
	reader      func(c *Channel, p *Packet, data []byte) bool
	reader2     func(data []byte, answer func(data []byte)) bool
	bslice.ByteSlice
}

// SetName set APIData name
func (a *APIData) SetName(name string) *APIData {
	a.name = name
	return a
}

// SetShort set APIData short name
func (a *APIData) SetShort(short string) *APIData {
	a.short = short
	return a
}

// SetLong set APIData long name (like description)
func (a *APIData) SetLong(long string) *APIData {
	a.long = long
	return a
}

// SetUsage set APIData usage text
func (a *APIData) SetUsage(usage string) *APIData {
	a.usage = usage
	return a
}

// SetReturn set APIData return description
func (a *APIData) SetReturn(ret string) *APIData {
	a.ret = ret
	return a
}

// SetCmd set APIData command number
func (a *APIData) SetCmd(cmd byte) *APIData {
	a.cmd = cmd
	return a
}

// SetConnectMode set APIData connect mode ( server|client|client&server )
func (a *APIData) SetConnectMode(connectMode APIconnectMode) *APIData {
	a.connectMode = connectMode
	return a
}

// SetAnswerMode set APIData answer mode (data|cmd|packet|none)
func (a *APIData) SetAnswerMode(answerMode APIanswerMode) *APIData {
	a.answerMode = answerMode
	return a
}

// SetReader set APIData reader
func (a *APIData) SetReader(reader func(c *Channel, p *Packet, data []byte) bool) *APIData {
	a.reader = reader
	return a
}

// SetReader2 set APIData second reader
func (a *APIData) SetReader2(reader2 func(data []byte, answer func(data []byte)) bool) *APIData {
	a.reader2 = reader2
	return a
}

// Name return APIData name
func (a APIData) Name() string { return a.name }

// Short return APIData short name
func (a APIData) Short() string { return a.short }

// Long return APIData long name
func (a APIData) Long() string { return a.long }

// Usage return APIData usage text
func (a APIData) Usage() string { return a.usage }

// Ret return APIData return mode
func (a APIData) Ret() string { return a.ret }

// Cmd return APIData cmd number
func (a APIData) Cmd() byte { return a.cmd }

// ExecMode return APIData exec mode
func (a APIData) ExecMode() (APIconnectMode, APIanswerMode) {
	return a.connectMode, a.answerMode
}

// Reader return APIData reader
func (a APIData) Reader(c *Channel, p *Packet, data []byte) bool {
	return a.reader(c, p, data)
}

// Reader2 return APIData second reader
func (a APIData) Reader2(data []byte, answer func(data []byte)) bool {
	return a.reader2(data, answer)
}

// MarshalBinary binary marshal APIData
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

// UnmarshalBinary binary unmarshal APIData
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
