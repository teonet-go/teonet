package teonet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"time"
)

var ErrIncorrectServerKey = errors.New("incorrect server key received")
var ErrIncorrectPublicKey = errors.New("incorrect public key received")

// Connect to teonet (client send to teonet auth server)
func (teo *Teonet) Connect(auth ...string) (err error) {

	teo.log.Println("connect to teonet")

	// TODO: Connect to auth https server and get auth ip:port to connect to
	//

	// Connect to trudp auth node
	ch, err := teo.trudp.Connect("localhost", 8000)
	if err != nil {
		return
	}

	var subs *subscribeData
	var chanW = make(chan []byte)
	teo.auth = teo.channels.new("", ch)
	subs = teo.subscribe(teo.auth, func(teo *Teonet, c *Channel, p *Packet, err error) bool {
		// Error processing
		if err != nil {
			teo.log.Printf("connect reader: got error from channel %s, error: %s", c, err)
			teo.unsubscribe(subs)
			teo.log.Println("disconnected from teonet")
			// Reconnect
			go func() {
				for {
					err := teo.Connect(auth...)
					if err == nil {
						break
					}
					time.Sleep(1 * time.Second)
				}
			}()
			return true
		}

		// Print received message
		// teo.log.Printf("connect reader: got from %s, \"%s\", len: %d, tt: %6.3fms\n",
		// 	c, p.Data, len(p.Data), float64(c.Triptime().Microseconds())/1000.0,
		// )

		// Unsubscribe
		chanW <- p.Data
		return true
	})

	// Connect data
	conIn := ConnectData{
		PubliKey:      teo.config.getPublicKey(),      // []byte("PublicKey"),
		Address:       []byte(teo.config.Address),     // []byte("Address"),
		ServerKey:     teo.config.ServerPublicKeyData, // []byte("ServerKey"),
		ServerAddress: nil,
	}

	// Encode
	data, err := conIn.MarshalBinary()
	if err != nil {
		// teo.log.Println("encode error:", err)
		return
	}
	// teo.log.Println("encoded ConnectData:", data, len(data))

	// Send to teoauth
	_, err = teo.trudp.Send(teo.auth.c, data)
	if err != nil {
		return
	}
	// teo.log.Println("send ConnectData to teoauth, id", id)

	// Receive connection answer
	data = <-chanW
	// teo.log.Println("got ansver from teoauth")

	// Decode
	var conOut ConnectData
	conOut.UnmarshalBinary(data)
	if err != nil {
		// teo.log.Println("decode error:", err)
		return
	}
	// teo.log.Printf("decoded ConnectData: %s\n", conOut)

	// Check server error
	if len(conOut.Err) > 0 {
		err = errors.New(string(conOut.Err))
		return
	}

	// Check received data
	if !reflect.DeepEqual(conOut.PubliKey, teo.GetPublicKey()) {
		err = ErrIncorrectPublicKey
		return
	}

	// Update config file
	addr := string(conOut.Address)
	teo.config.ServerPublicKeyData = conOut.ServerKey
	teo.config.Address = addr
	teo.config.save()

	teo.log.Println("connected to teonet")
	teo.log.Printf("teonet address: %s\n", conOut.Address)

	teo.Connected(teo.auth, addr)

	return
}

// ConnectProcess check connection to teonet (auth server receive connection
// data from client) and send answer to client
func (teo Teonet) ConnectProcess(c *Channel, data []byte) (err error) {
	// Decode
	var con ConnectData
	con.UnmarshalBinary(data)
	if err != nil {
		teo.log.Println("decode error:", err)
		return
	}
	// teo.log.Printf("decoded ConnectData: %s\n", con)

	sendAnswer := func() (err error) {
		// Encode
		data, err = con.MarshalBinary()
		if err != nil {
			teo.log.Println("marshal error:", err)
			return
		}

		// Send snswer
		_, err = c.c.SendAnswer(data)
		if err != nil {
			teo.log.Println("send answer error:", err)
		}
		return
	}

	// Check server key and set it if empty
	if len(con.ServerKey) == 0 {
		con.ServerKey = teo.GetPublicKey()
	} else if !reflect.DeepEqual(con.ServerKey, teo.GetPublicKey()) {
		con.Err = []byte(ErrIncorrectServerKey.Error())
		err = sendAnswer()
		return
	}

	// Set server Address
	con.ServerAddress = []byte(teo.GetAddress())

	// TODO: check client Address
	if len(con.Address) == 0 {
		var addr string
		addr, err = teo.MakeAddress(con.PubliKey)
		if err != nil {
			teo.log.Println("make client address error:", err)
			return
		}
		con.Address = []byte(addr)
	}

	err = sendAnswer()
	if err != nil {
		return
	}

	// Add to clients map
	teo.Connected(c, string(con.Address))
	teo.log.Println("client connected:", string(con.Address))

	return
}

// Connected set address to channel and add channel to channels list
func (teo Teonet) Connected(c *Channel, addr string) {
	c.a = addr
	teo.channels.add(c)
}

// ConnectData teonet connect data
type ConnectData struct {
	PubliKey      []byte // Client public key (generated from private key)
	Address       []byte // Client address (received after connect if empty)
	ServerKey     []byte // Server public key (send if exists or received in connect if empty)
	ServerAddress []byte // Server address (received after connect)
	Err           []byte // Error of connect data processing
}

func (c ConnectData) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	writeSlice := func(data []byte) {
		binary.Write(buf, binary.LittleEndian, uint16(len(data)))
		binary.Write(buf, binary.LittleEndian, data)
	}

	writeSlice(c.PubliKey)
	writeSlice(c.Address)
	writeSlice(c.ServerKey)
	writeSlice(c.ServerAddress)
	writeSlice(c.Err)

	data = buf.Bytes()

	return
}

func (c *ConnectData) UnmarshalBinary(data []byte) (err error) {

	buf := bytes.NewBuffer(data)

	readSlice := func() (data []byte, err error) {
		var l uint16
		err = binary.Read(buf, binary.LittleEndian, &l)
		if err != nil {
			return
		}
		data = make([]byte, l)
		binary.Read(buf, binary.LittleEndian, data)
		if err != nil {
			return
		}
		return
	}

	c.PubliKey, err = readSlice()
	if err != nil {
		return
	}
	c.Address, err = readSlice()
	if err != nil {
		return
	}
	c.ServerKey, err = readSlice()
	if err != nil {
		return
	}
	c.ServerAddress, err = readSlice()
	if err != nil {
		return
	}
	c.Err, err = readSlice()
	if err != nil {
		return
	}

	return
}

func (c ConnectData) String() string {
	return fmt.Sprintf("len: %d\nkey: %x\naddress: %s\nserver key: %x\nserver address: %s\nerror: %s",
		len(c.PubliKey)+len(c.Address)+len(c.ServerKey)+len(c.ServerAddress)+len(c.Err),
		c.PubliKey,
		c.Address,
		c.ServerKey,
		c.ServerAddress,
		c.Err,
	)
}
