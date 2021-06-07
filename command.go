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

	cmd = &Command{teo: teo}
	switch len(attr) {
	case 1:
		switch c := attr[0].(type) {
		case []byte:
			cmd.UnmarshalBinary(c)
		default:
			panic("wrong data attribute")
		}
	case 2:
		// command
		switch c := attr[0].(type) {
		case AuthCmd:
			cmd.Cmd = byte(c)
		case byte:
			cmd.Cmd = c
		case int:
			cmd.Cmd = byte(c)
		default:
			panic("wrong cmd attribute")
		}
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
	return
}

type Command struct {
	Cmd  byte
	Data []byte
	teo  *Teonet
}

func (c Command) Bytes() (data []byte) {
	data, _ = c.MarshalBinary()
	return
}

func (c Command) Send(channel *Channel) (id uint32, err error) {
	return channel.Send(c.Bytes())
}

func (c Command) SendNoWait(channel *Channel) (id uint32, err error) {
	return channel.SendNoWait(c.Bytes())
}

func (c Command) SendTo(addr string) (id uint32, err error) {
	return c.teo.SendTo(addr, c.Bytes())
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

// NewCommandInterface create teonet client with command interfeice commected
func NewCommandInterface(appName string, attr ...interface{}) (teo *TeonetCommand, err error) {
	t, err := New(appName, attr...)
	if err != nil {
		return
	}
	teo = commandInterface(t)
	return
}

// CommandInterface get teonet command interface
// func (t *Teonet) CommandInterface() (teo *TeonetCommand) {
// 	teo = commandInterface(t)
// 	return
// }

// commandInterface get teonet command interface
func commandInterface(t *Teonet) (teo *TeonetCommand) {
	teo = &TeonetCommand{t, t.newWaitFrom()}
	teo.addClientReader(func(t *Teonet, c *Channel, p *Packet, err error) (ret bool) {
		return teo.readerCommand(c, p, err)
	})
	return
}

// TeonetCommand is teonet command interface
type TeonetCommand struct {
	*Teonet
	wcom *waitCommand
}

// readerCommand is Main teonet reader running in teonet command interface mode.
// This reader process received waitFrom commands
func (teo *TeonetCommand) readerCommand(c *Channel, p *Packet, err error) (ret bool) {
	// Process waitFrom packets
	if err == nil {
		if teo.wcom.check(p) > 0 {
			return true
		}
	}
	return
}

type ApiInterface interface {
	ProcessPacket(p interface{})
}

// setApiReader sets teonet reader. This reader process received API commands
func (teo *Teonet) setApiReader(api ApiInterface) {
	if api == nil {
		return
	}
	teo.addClientReader(func(teo *Teonet, c *Channel, p *Packet, err error) (ret bool) {
		// Process API commands
		if err == nil {
			api.ProcessPacket(p.setCommandMode())
		}
		return
	})
}

func (teo TeonetCommand) SendAnswer(i interface{}, cmd byte, data []byte) (n int, err error) {
	pac := i.(*Packet)
	return teo.SendTo(pac.From(), cmd, data)
}

func (teo TeonetCommand) SendTo(addr string, cmd byte, data []byte) (n int, err error) {
	id, err := teo.Command(cmd, data).SendTo(addr)
	n = int(id)
	return
}
