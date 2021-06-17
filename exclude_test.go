// Test of ConnectIpPort.exclude function
package teonet

import (
	"errors"
	"fmt"
	"testing"
)

func TestExcludeIPs(t *testing.T) {

	var ErrWrong = errors.New("wrong execution")
	var con ConnectIpPort

	nodeAddr := []NodeAddr{
		{"129.0.0.1", 121},
		{"127.0.0.1", 122},
		{"128.0.0.1", 123},
		{"129.0.0.1", 124},
		{"128.0.0.1", 125},
		{"127.0.0.1", 126},
	}
	fmt.Println(nodeAddr)

	nodeAddr = con.exclude(nodeAddr, "128.0.0.1")
	fmt.Println(nodeAddr)
	if len(nodeAddr) != 4 {
		t.Error(ErrWrong)
		return
	}

	nodeAddr = con.exclude(nodeAddr, "127.0.0.1")
	fmt.Println(nodeAddr)
	if len(nodeAddr) != 2 {
		t.Error(ErrWrong)
		return
	}

	nodeAddr = con.exclude(nodeAddr, "130.0.0.1")
	fmt.Println(nodeAddr)
	if len(nodeAddr) != 2 {
		t.Error(ErrWrong)
		return
	}

	nodeAddr = con.exclude(nodeAddr, "129.0.0.1")
	fmt.Println(nodeAddr)
	if len(nodeAddr) != 0 {
		t.Error(ErrWrong)
		return
	}
}
