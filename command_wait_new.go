package teonet

import (
	"encoding/binary"
	"log"
	"time"

	"github.com/kirill-scherba/trudp"
)

type CheckDataFunc func(data []byte) (ok bool)
type WaitData []byte

// WaitFrom wait answer from addres. Attr is additional attributes by type:
//   byte or int: wait command number in answer
//   uint32: wait packet id in answer
//   func([]byte)bool: check packet data with callback, data without command and id
//   time.Duration: wait timeout (default 5 sec)
//
//   answer packet data structure: [cmd][id][data] it depend of service api
func (teo *Teonet) WaitFrom(from string, attr ...interface{}) (data []byte, err error) {

	attr = append(attr, true)
	wr := teo.MakeWaitReader(attr...)
	if err != nil {
		return
	}

	scr, err := teo.Subscribe(from, wr.Reader())
	if err != nil {
		return
	}

	data, err = teo.WaitReaderAnswer(wr.Wait(), wr.Timeout())

	teo.Unsubscribe(scr)

	return
}

// WaitReaderAnswer wait data from reader, return received data or error on timeout
func (teo *Teonet) WaitReaderAnswer(wait chan WaitData, timeout time.Duration) (data []byte, err error) {
	select {
	case data = <-wait:
	case <-time.After(timeout):
		err = ErrTimeout
	}
	return
}

type WaitReader struct {
	wait    chan WaitData
	reader  func(c *Channel, p *Packet, err error) (processed bool)
	timeout time.Duration
}

// MakeWaitReader create reader, wait channel and timeout from attr:
//   byte or int: wait command number in answer
//   uint32: wait packet id in answer
//   func([]byte)bool: check packet data with callback, data without command and id
//   time.Duration: wait timeout (default 5 sec)
//   bool: created wait channel and send data to channel if true
//
//   answer packet data structure: [cmd][id][data] it depend of service api
func (teo *Teonet) MakeWaitReader(attr ...interface{}) (wr *WaitReader) {

	wr = new(WaitReader)

	// Parse attr
	const (
		validCmd byte = 1 << iota
		validID
		validF
	)
	var param struct {
		cmd   byte
		id    uint32
		f     CheckDataFunc
		check byte
		wait  bool
	}
	wr.timeout = trudp.ClientConnectTimeout
	for _, a := range attr {
		switch v := a.(type) {

		case byte:
			param.cmd = v
			param.check |= validCmd

		case int:
			param.cmd = byte(v)
			param.check |= validCmd

		case uint32:
			param.id = v
			param.check |= validID

		case func([]byte) bool:
			param.f = v
			param.check |= validF

		case bool:
			param.wait = v

		case time.Duration:
			wr.timeout = v

		default:
			log.Panicf("wrong reader attribute with type %T\n", v)
		}
	}

	wr.reader = func(c *Channel, p *Packet, err error) (processed bool) {

		if err != nil {
			return
		}

		var idx = 0

		// Check Command
		if param.check&validCmd > 0 {
			cmd := p.Data()[idx]
			if cmd != param.cmd {
				return
			}
			idx += 1
		}

		// Check ID
		if param.check&validID > 0 {
			if len(p.Data()[idx:]) < 4 {
				return
			}
			id := binary.LittleEndian.Uint32(p.Data()[idx:])
			if id != param.id {
				return
			}
			idx += 4
		}

		// Check data func
		if param.check&validF > 0 {
			if !param.f(p.Data()[idx:]) {
				return
			}
		}

		// Valid packet
		processed = true

		// Send answer to wait channel
		if param.wait {
			wr.wait <- p.Data()[idx:]
		}

		return
	}

	if param.wait {
		wr.wait = make(chan WaitData)
	}

	return
}

func (wr WaitReader) Wait() chan WaitData { return wr.wait }
func (wr WaitReader) Reader() func(c *Channel, p *Packet, err error) (processed bool) {
	return wr.reader
}
func (wr WaitReader) Timeout() time.Duration { return wr.timeout }

func (teo *Teonet) MakeWaitAttr() *WaitAttr {
	return new(WaitAttr)
}

type WaitAttr struct {
	attr []interface{}
}

func (w *WaitAttr) Cmd(cmd byte) *WaitAttr {
	w.attr = append(w.attr, cmd)
	return w
}
func (w *WaitAttr) ID(id uint32) *WaitAttr {
	w.attr = append(w.attr, id)
	return w
}
func (w *WaitAttr) Func(f func([]byte) bool) *WaitAttr {
	w.attr = append(w.attr, f)
	return w
}
func (w *WaitAttr) Timeout(t time.Duration) *WaitAttr {
	w.attr = append(w.attr, t)
	return w
}
func (w *WaitAttr) GetAttr() []interface{} {
	return w.attr
}