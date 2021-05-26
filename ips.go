// Copyright 2021 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Connect to peer module

package teonet

import (
	"net"
	"strings"
)

const IPv6Allow = true

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
