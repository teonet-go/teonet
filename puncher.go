// Copyright 2021-22 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet Get IPs and Puncher module

package teonet

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kirill-scherba/tru"
)

// Allow IPv6 connection between peers
const IPv6Allow = true

// puncher struct and methods receiver
type puncher struct {
	tru *tru.Tru
	m   map[string]*PuncherData
	sync.RWMutex
}

// PuncherData is puncher data struct
type PuncherData struct {
	wait *chan *net.UDPAddr
}

// IPs struct contain peers local and global IPs and ports
type IPs struct {
	LocalIPs  []string
	LocalPort uint32
	IP        string
	Port      uint32
}

// getIPs return string slice with this host local IPs address
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
			// Check ipv6 address, add [ ... ] if ipv6 allowed and
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

// newPuncher create new puncher and set punch callback to tru
func (teo *Teonet) newPuncher() {
	if teo.tru == nil {
		panic("trudp should be Init befor call to newPuncher()")
	}
	teo.puncher = &puncher{tru: teo.tru, m: make(map[string]*PuncherData)}

	// Connect puncer to TRU - set punch callback
	teo.tru.SetPunchCb(func(addr net.Addr, data []byte) {
		log.Debug.Printf("puncher get %s from %s\n", string(data), addr.String())
		teo.puncher.callback(data, addr.(*net.UDPAddr))
	})
}

// subscribe to puncher - add to map
func (p *puncher) subscribe(key string, punch *PuncherData) {
	p.Lock()
	defer p.Unlock()
	p.m[key] = punch
}

// unsubscribe from puncher - delete from map
func (p *puncher) unsubscribe(key string) {
	p.Lock()
	defer p.Unlock()
	delete(p.m, key)
}

// get key from map
func (p *puncher) get(key string) (punch *PuncherData, ok bool) {
	p.RLock()
	defer p.RUnlock()
	punch, ok = p.m[key]
	return
}

// callback process received puch packet
func (p *puncher) callback(data []byte, addr *net.UDPAddr) (ok bool) {
	key := string(data)
	punch, ok := p.get(key)
	if ok {
		p.unsubscribe(key)
		*punch.wait <- addr
	}
	return
}

// send puncher key to list of IP:Port
func (p *puncher) send(key string, ips IPs) (err error) {

	sendKey := func(ip string, port uint32) (err error) {
		addr := ip + ":" + strconv.Itoa(int(port))
		dst, err := p.tru.WriteToPunch([]byte(key), addr)
		log.Debug.Printf("puncher send %s to %s\n", key, dst.String())
		return
	}
	for i := range ips.LocalIPs {
		sendKey(ips.LocalIPs[i], ips.LocalPort)
	}
	sendKey(ips.IP, ips.Port)

	return
}

// Punch client ip:ports (send udp packets to received IPs)
//   delays parameter is start punch delay
func (p *puncher) punch(key string, ips IPs, stop func() bool, delays ...time.Duration) {
	go func() {
		if len(delays) > 0 {
			time.Sleep(delays[0])
		}
		for i := 0; i < 5; /* 15 */ i++ {
			if stop() {
				break
			}
			p.send(key, ips)
			time.Sleep(time.Duration(((i + 1) * 30)) * time.Millisecond)
		}
	}()
}
