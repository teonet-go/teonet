// Copyright 2029-2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet wait command module.

package teonet

/*

// waitFromNew initialize new Wait command module
func (teo *Teonet) NewWaitFrom() (wcom *waitCommand) {
	wcom = &waitCommand{}
	wcom.m = make(map[string][]*waitFromRequest)
	wcom.teo = teo
	return
}

// waitCommand is wait command eeiver
type waitCommand struct {
	m map[string][]*waitFromRequest // 'wait command from' requests map
	sync.RWMutex
	teo *Teonet
}

// waitFromRequest 'wait command from' request
type waitFromRequest struct {
	from      string           // waiting from
	cmd       byte             // waiting comand
	ch        ChanWaitFromData // return channel
	checkFunc checkDataFunc    // check data func
}

// ChanWaitFromData 'wait command from' return channel
// type ChanWaitFromData chan *WaitFromData
type ChanWaitFromData chan *struct {
	Data []byte
	Err  error
}

//type ChanWaitFromDataRead <-chan *WaitFromData

// WaitFromData
// type WaitFromData struct {
// 	Data []byte
// 	Err  error
// }

func (wcom *waitCommand) Reader(c *Channel, p *Packet, err error) (ret bool) {
	// Process waitFrom packets
	if err == nil {
		if wcom.check(p) > 0 {
			return true
		}
	}
	return
}

// add adds 'wait command from' request
func (wcom *waitCommand) add(from string, cmd byte, ch ChanWaitFromData, f checkDataFunc) (wfr *waitFromRequest) {
	wcom.Lock()
	defer wcom.Unlock()

	key := wcom.makeKey(from, cmd)
	wcomRequestAr, ok := wcom.m[key]
	wfr = &waitFromRequest{from, cmd, ch, f}
	if !ok {
		wcom.m[key] = []*waitFromRequest{wfr}
		return
	}
	wcom.m[key] = append(wcomRequestAr, wfr)
	return
}

// exists checks if waitFromRequest exists
func (wcom *waitCommand) exists(wfr *waitFromRequest, removes ...bool) (found bool) {
	var remove bool
	if len(removes) > 0 {
		remove = removes[0]
	}
	if remove {
		wcom.Lock()
		defer wcom.Unlock()
	} else {
		wcom.RLock()
		defer wcom.RUnlock()
	}

	key := wcom.makeKey(wfr.from, wfr.cmd)
	wcar, ok := wcom.m[key]
	if !ok {
		return
	}
	for idx, w := range wcar {
		if w == wfr {
			// remove element if second parameter of this function == true
			if remove {
				wcar = append(wcar[:idx], wcar[idx+1:]...)
				if len(wcar) == 0 {
					delete(wcom.m, key)
				}
			}
			return true
		}
	}
	return
}

// remove removes wait command from request
func (wcom *waitCommand) remove(wfr *waitFromRequest) {
	wcom.exists(wfr, true)
}

// check if wait command for received command exists in wait command map and
// send receiving data to wait command channel if so
func (wcom *waitCommand) check(p *Packet) (processed int) {

	wcom.Lock()
	defer wcom.Unlock()

	var cmd *Command
	if p.commandMode {
		cmd = wcom.teo.Command(p.Cmd(), p.Data())
	} else {
		cmd = wcom.teo.Command(p.Data())
	}

	key := wcom.makeKey(p.From(), cmd.Cmd)
	wcar, ok := wcom.m[key]
	if !ok {
		return
	}

	for {
		for idx, w := range wcar {
			if w.checkFunc != nil {
				if !w.checkFunc(cmd.Data) {
					continue
				}
			}
			// Found (delete it from slice)
			wcar = append(wcar[:idx], wcar[idx+1:]...)

			// Send data to wait channel
			w.ch <- &struct {
				Data []byte
				Err  error
			}{cmd.Data, nil}
			close(w.ch)
			processed++

			// Delete key if slice empty
			if len(wcar) == 0 {
				delete(wcom.m, key)
				return
			}
			break
		}
	}
}

// makeKey make wait command map key
func (wcom *waitCommand) makeKey(from string, cmd byte) string {
	return from + ":" + strconv.Itoa(int(cmd))
}

type checkDataFunc func([]byte) bool

// WaitFrom wait receiving data from peer. The third function parameter is
// timeout. It may be omitted or contain timeout time of time. Duration type.
// If timeout parameter is omitted than default timeout value sets to 2 second.
// Next parameter is checkDataFunc func([]byte) bool. This function calls to
// check packet data and returns true if packet data valid. This parameter may
// be ommited too.
func (wcom *waitCommand) WaitFrom(from string, cmd byte, attr ...interface{}) (data []byte, err error) {
	res := <-wcom.waitFrom(from, cmd, attr)
	return res.Data, res.Err
}

func (wcom *waitCommand) waitFrom(from string, cmd byte, attr ...interface{}) <-chan *struct {
	Data []byte
	Err  error
} {
	// Parameters definition
	var checkFunc checkDataFunc
	timeout := 5 * time.Second
	for i := range attr {
		switch v := attr[i].(type) {
		case time.Duration:
			timeout = v
		case func([]byte) bool:
			checkFunc = v
		}
	}

	// Create channel, add wait parameter and wait timeout
	ch := make(chan *struct {
		Data []byte
		Err  error
	})

	go func() {
		wfr := wcom.add(from, cmd, ch, checkFunc)
		time.Sleep(timeout)
		if wcom.exists(wfr) {
			ch <- &struct {
				Data []byte
				Err  error
			}{nil, errors.New("timeout")}
			wcom.remove(wfr)
		}
	}()

	return ch
}

*/

// TODO: check replacing function body to 'return teo.wcom.WaitFrom(from, cmd, attr...)
// func (teo *TeonetCommand) WaitFrom(from string, cmd byte, attr ...interface{}) <-chan *struct {
// 	Data []byte
// 	Err  error
// } {
// 	return teo.wcom.waitFrom(from, cmd, attr...)
// }
