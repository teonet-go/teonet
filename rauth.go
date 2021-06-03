package teonet

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/kirill-scherba/teonet-go/teolog/teolog"
)

// Nodes get auth nodes by URL
func Nodes(url string) (ret *nodes, err error) {
	resp, err := http.Get(url)
	if err != nil {
		teolog.Log(teolog.ERROR, "HTTP", "server", err)
		return
	}
	// log.Println(resp)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		teolog.Log(teolog.ERROR, "HTTP", "server", err)
		return
	}

	dst := make([]byte, hex.DecodedLen(len(body)))
	n, err := hex.Decode(dst, body)
	if err != nil {
		log.Fatal(err)
	}

	ret = new(nodes)
	ret.UnmarshalBinary(dst[:n])
	return
}

type nodes struct {
	byteSlice
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

// byteSlice help binary marshal/ubmarshal byte slice
type byteSlice struct{}

func (b byteSlice) WriteSlice(buf *bytes.Buffer, data []byte) (err error) {
	if err = binary.Write(buf, binary.LittleEndian, uint16(len(data))); err != nil {
		return
	}
	err = binary.Write(buf, binary.LittleEndian, data)
	return
}

func (b byteSlice) ReadSlice(buf *bytes.Buffer) (data []byte, err error) {
	var l uint16
	if err = binary.Read(buf, binary.LittleEndian, &l); err != nil {
		return
	}
	data = make([]byte, l)
	err = binary.Read(buf, binary.LittleEndian, data)
	return
}

func (b byteSlice) ReadString(buf *bytes.Buffer) (data string, err error) {
	d, err := b.ReadSlice(buf)
	if err != nil {
		return
	}
	data = string(d)
	return
}

func (b byteSlice) WriteStringSlice(buf *bytes.Buffer, data []string) (err error) {
	if err = binary.Write(buf, binary.LittleEndian, uint16(len(data))); err != nil {
		return
	}
	for i := range data {
		if err = b.WriteSlice(buf, []byte(data[i])); err != nil {
			return
		}
	}

	return
}

func (b byteSlice) ReadStringSlice(buf *bytes.Buffer) (data []string, err error) {
	var l uint16
	if err = binary.Read(buf, binary.LittleEndian, &l); err != nil {
		return
	}
	for i := 0; i < int(l); i++ {
		var d []byte
		if d, err = b.ReadSlice(buf); err != nil {
			return
		}
		data = append(data, string(d))
	}
	return
}
