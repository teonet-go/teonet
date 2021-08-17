// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Connect to peer IPs and Puncer module

package teonet

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kirill-scherba/trudp"
)

const IPv6Allow = false

type connectModeT byte

const (
	clientMode connectModeT = iota
	serverMode
)

// getIPs return string slice with local IP address of this host
func (teo Teonet) getIPs() (ips []string, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			a := ip.String()
			// Check ipv6 address add [] if ipv6 allowed and
			// skip this address if ipv6 not allowed
			if strings.IndexByte(a, ':') >= 0 {
				if !IPv6Allow {
					continue
				}
				a = "[" + a + "]"
			}
			ips = append(ips, a)
		}
	}
	return
}

func (teo *Teonet) newPuncher() {
	if teo.trudp == nil {
		panic("trudp should be Init befor call to newPuncher()")
	}
	teo.puncher = &puncher{trudp: teo.trudp, m: make(map[string]*PuncherData)}
	teo.trudp.SetPunchCb(func(data []byte, addr *net.UDPAddr) bool {
		return teo.puncher.callback(data, addr)
	})
}

type puncher struct {
	trudp *trudp.Trudp
	m     map[string]*PuncherData
	sync.RWMutex
}

type IPs struct {
	LocalIPs  []string
	LocalPort uint32
	IP        string
	Port      uint32
}

type PuncherData struct {
	wait *chan *net.UDPAddr
}

// type waitChan chan *net.UDPAddr

func (p *puncher) subscribe(key string, punch *PuncherData) {
	p.Lock()
	defer p.Unlock()
	p.m[key] = punch
	// fmt.Println("puncher subscribe to:", key)
}

func (p *puncher) unsubscribe(key string) {
	p.Lock()
	defer p.Unlock()
	delete(p.m, key)
}

func (p *puncher) get(key string) (punch *PuncherData, ok bool) {
	p.RLock()
	defer p.RUnlock()
	punch, ok = p.m[key]
	return
}

func (p *puncher) callback(data []byte, addr *net.UDPAddr) (ok bool) {
	key := string(data)
	punch, ok := p.get(key)
	// fmt.Println(">>>>>>>>>>> punch callback, key:", key, "from:", addr, "get-ok:", ok)
	if ok {
		p.unsubscribe(key)
		*punch.wait <- addr
	}
	return
}

func (p *puncher) send(key string, ips IPs) (err error) {

	sendKey := func(ip string, port uint32) (int, error) {
		addr := ip + ":" + strconv.Itoa(int(port))
		dst, err := net.ResolveUDPAddr(trudp.Network, addr)
		if err != nil {
			return 0, err
		}
		return p.trudp.WriteTo(dst, []byte(key))
	}
	for i := range ips.LocalIPs {
		sendKey(ips.LocalIPs[i], ips.LocalPort)
	}
	sendKey(ips.IP, ips.Port)

	return
}

// Punch client ip:ports (send udp packets to received IPs)
// mode: true - server to client mode; false - client to server mode
func (p *puncher) punch(key string, ips IPs, stop func() bool, mode connectModeT, delays ...time.Duration) {
	go func() {
		if len(delays) > 0 {
			time.Sleep(delays[0])
		}
		for i := 0; i < 15; i++ {
			if stop() {
				break
			}
			if mode == clientMode {
				key = trudp.PunchPrefix + key
			}
			p.send(key, ips)
			time.Sleep(time.Duration(((i + 1) * 30)) * time.Millisecond)
		}
	}()
}
