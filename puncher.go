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

	"github.com/teonet-go/tru"
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
			a, ok := teo.safeIPv6(a)
			if !IPv6Allow && ok {
				continue
			}
			ips = append(ips, a)
		}
	}
	return
}

// safeIPv6 check ipv6 address and add [ ... ] to it, set ok to true if address
// is IPv6
func (teo Teonet) safeIPv6(ipin string) (ipout string, ok bool) {
	if strings.IndexByte(ipin, ':') >= 0 {
		ok = true
		ipout = "[" + ipin + "]"
	} else {
		ipout = ipin
	}

	return
}

// newPuncher create new puncher and set punch callback to tru
func (teo *Teonet) newPuncher() {
	if teo.tru == nil {
		panic("trudp should be Init befor call to newPuncher()")
	}
	teo.puncher = &puncher{tru: teo.tru, m: make(map[string]*PuncherData)}

	// Connect puncher to TRU - set punch callback
	teo.tru.SetPunchCb(func(addr net.Addr, data []byte) {
		log.Debugv.Printf("puncher get %s from %s\n", string(data[:6]), addr.String())
		teo.puncher.callback(data, addr.(*net.UDPAddr))
	})
}

// subscribe to puncher - add to map
func (p *puncher) subscribe(key string, punch *PuncherData) {
	p.Lock()
	defer p.Unlock()
	p.m[key] = punch
}

// unsubscribe from puncher - delete from map, return ok = true and PuncherData
// if subscribe exists
func (p *puncher) unsubscribe(key string) (punch *PuncherData, ok bool) {
	p.Lock()
	defer p.Unlock()
	punch, ok = p.m[key]
	if ok {
		delete(p.m, key)
	}
	return
}

// callback process received puch packet
func (p *puncher) callback(data []byte, addr *net.UDPAddr) (ok bool) {
	var punch *PuncherData
	if punch, ok = p.unsubscribe(string(data)); ok {
		*punch.wait <- addr
	}
	return
}

// send puncher key to list of IP:Port
func (p *puncher) send(key string, ips IPs, stop ...func() bool) (err error) {

	sendKey := func(ip string, port uint32) (err error) {
		addr := ip + ":" + strconv.Itoa(int(port))
		dst, err := p.tru.WriteToPunch([]byte(key), addr)
		log.Debugv.Printf("puncher send %s to %s\n", key[:6], dst.String())
		return
	}
	for i := range ips.LocalIPs {
		if len(stop) > 0 && stop[0]() {
			return
		}
		sendKey(ips.LocalIPs[i], ips.LocalPort)
	}
	sendKey(ips.IP, ips.Port)

	return
}

// Punch client ip:ports (send udp packets to received IPs)
//
//	delays parameter is start punch delay
func (p *puncher) punch(key string, ips IPs, stop func() bool, delays ...time.Duration) {
	go func() {
		if len(delays) > 0 {
			time.Sleep(delays[0])
		}
		for i := 0; i < 5 && !stop(); i++ {
			p.send(key, ips, stop)
			time.Sleep(time.Duration(((i + 1) * 30)) * time.Millisecond)
		}
	}()
}
