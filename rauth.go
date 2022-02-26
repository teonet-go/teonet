package teonet

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/kirill-scherba/bslice"
)

// Nodes get auth nodes by URL
func Nodes(url string) (ret *nodes, err error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Error.Println("HTTP", "server", err)
		return
	}
	// log.Println(resp)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error.Println("HTTP", "server", err)
		return
	}

	dst := make([]byte, hex.DecodedLen(len(body)))
	n, err := hex.Decode(dst, body)
	if err != nil {
		log.Error.Println("HTTP", "server can't decode answer, error:", err)
		return
	}

	ret = new(nodes)
	ret.UnmarshalBinary(dst[:n])
	return
}

type nodes struct {
	bslice.ByteSlice
	address []NodeAddr
}

type NodeAddr struct {
	IP   string
	Port uint32
}

func (r nodes) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	l := uint32(len(r.address))
	binary.Write(buf, binary.LittleEndian, l)
	for i := 0; i < int(l); i++ {
		r.WriteSlice(buf, []byte(r.address[i].IP))
		binary.Write(buf, binary.LittleEndian, r.address[i].Port)
	}

	data = buf.Bytes()
	return
}

func (r *nodes) UnmarshalBinary(data []byte) (err error) {
	buf := bytes.NewBuffer(data)

	var l uint32
	if err = binary.Read(buf, binary.LittleEndian, &l); err != nil {
		return
	}
	r.address = make([]NodeAddr, l)
	for i := 0; i < int(l); i++ {
		if r.address[i].IP, err = r.ReadString(buf); err != nil {
			return
		}
		if err = binary.Read(buf, binary.LittleEndian, &r.address[i].Port); err != nil {
			return
		}
	}

	return
}

func (r nodes) String() (s string) {
	for i := range r.address {
		if i != 0 {
			s += "\n"
		}
		s += fmt.Sprintf("%s:%d", r.address[i].IP, r.address[i].Port)
	}
	return
}

// Slice return nodes address slice
func (r nodes) Slice() []NodeAddr {
	return r.address
}
