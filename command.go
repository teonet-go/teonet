// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet command module

package teonet

import (
	"errors"
)

var ErrCommandTooShort = errors.New("command packet too short")

func (teo *Teonet) Command(attr ...interface{}) (cmd *Command) {
	switch len(attr) {
	case 1:
		cmd = new(Command)
		cmd.UnmarshalBinary(attr[0].([]byte))
	case 2:
		cmd = new(Command)
		// comand
		switch c := attr[0].(type) {
		case AuthCmd:
			cmd.Cmd = byte(c)
		default:
			cmd.Cmd = c.(byte)
		}
		// cmd.Cmd = byte(attr[0].(AuthCmd))
		// data
		switch d := attr[1].(type) {
		case []byte:
			cmd.Data = d
		case string:
			cmd.Data = []byte(d)
		default:
			panic("wrong data attribute")
		}
	}
	// cmd.teo = teo
	return
}

type Command struct {
	Cmd  byte
	Data []byte
	// teo  *Teonet
}

func (c Command) Bytes() (data []byte) {
	data, _ = c.MarshalBinary()
	return
}

func (c Command) Send(channel *Channel) (id uint32, err error) {
	// return c.teo.trudp.Send(channel.c, c.Bytes())
	return channel.Send(c.Bytes())
}

func (c Command) SendAnswer(channel *Channel) (id uint32, err error) {
	// return c.teo.trudp.SendAnswer(channel.c, c.Bytes())
	return channel.SendAnswer(c.Bytes())
}

func (c Command) MarshalBinary() (data []byte, err error) {
	data = append([]byte{c.Cmd}, c.Data...)
	return
}

func (c *Command) UnmarshalBinary(data []byte) (err error) {
	if len(data) < 1 {
		return ErrCommandTooShort
	}
	c.Cmd = data[0]
	c.Data = data[1:]
	return
}
