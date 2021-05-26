package teonet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"time"
)

// Teoauth commands
const (
	// CmdConnect send <cmd byte, data ConnectData> to teonet auth server to
	// connect to teonet; receive <cmd byte, data ConnectData> from teonet auth
	// server when connection established
	CmdConnect AuthCmd = iota

	// CmdConnectTo send <cmd byte, data ConnectToData> to teonet auth server to
	// connect to peer
	CmdConnectTo

	// CmdConnectToPeer command send by teonet auth to server to receive
	// connection from client
	CmdConnectToPeer
)

type AuthCmd byte

// Connet errors
var ErrIncorrectServerKey = errors.New("incorrect server key received")
var ErrIncorrectPublicKey = errors.New("incorrect public key received")

// Connect to errors

// Connect to teonet (client send request to teonet auth server):
// Client call Connect (and wait answer inside Connect function) -> Server call
// ConnectProcess -> Client got answer (inside Connect function) and set teonet
// Connected (create teonet channel)
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
	teo.auth = teo.channels.new(ch)
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

		// Process commands from teonet server
		cmd := teo.Command(p.Data)
		switch AuthCmd(cmd.Cmd) {
		case CmdConnect:
			chanW <- cmd.Data
		case CmdConnectTo:
			go teo.connectToAnswerProcess(cmd.Data)
		case CmdConnectToPeer:
			teo.log.Println("got CmdConnectToPeer command, data len:", len(cmd.Data))
		default:
			teo.log.Println("not defined command", cmd.Cmd)
		}

		return true
	})

	// Connect data
	conIn := ConnectData{
		PubliKey:      teo.config.getPublicKey(),      // []byte("PublicKey"),
		Address:       []byte(teo.config.Address),     // []byte("Address"),
		ServerKey:     teo.config.ServerPublicKeyData, // []byte("ServerKey"),
		ServerAddress: nil,
	}

	// Marshal data
	data, err := conIn.MarshalBinary()
	if err != nil {
		// teo.log.Println("encode error:", err)
		return
	}
	// teo.log.Println("encoded ConnectData:", data, len(data))

	// Send to teoauth
	// cmd := teo.Command(CmdConnect, data)
	// _, err = teo.trudp.Send(teo.auth.c, cmd.Bytes())
	_, err = teo.Command(CmdConnect, data).Send(teo.auth)
	if err != nil {
		return
	}
	// teo.log.Println("send ConnectData to teoauth, id", id)

	// Wait Connect answer data
	data = <-chanW
	// teo.log.Println("got ansver from teoauth")

	// Unmarshal data
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

	teo.Connected(teo.auth, string(conOut.ServerAddress))

	return
}

// ConnectProcess check connection to teonet and send answer to client (auth
// server receive connection data from client)
func (teo Teonet) ConnectProcess(c *Channel, data []byte) (err error) {
	// Unmarshal data
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
		// cmd := teo.Command(CmdConnect, data)
		// _, err = c.c.SendAnswer(cmd.Bytes())
		_, err = teo.Command(CmdConnect, data).SendAnswer(c)
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
	// teo.log.Println("client connected:", string(con.Address))

	return
}

// ConnectTo connect to any teonet Peer(client or server) by address (client
// sent request to teonet auth server):
// Client call ConnectTo wich send request to teonet auth server and wait
// function connectToAnswerProcess called -> Server call ConnectToProcess send
// infor to Peer and send answer to client (connectToAnswerProcess func called
// on client side when answer received) -> Client connect to Peer and send
// clients teonet address to it
func (teo Teonet) ConnectTo(addr string) (err error) {
	// TODO: check local connection exists

	// Connect data
	conIn := ConnectToData{
		Addr: addr,
	}
	data, _ := conIn.MarshalBinary()
	// Send command to teonet
	teo.Command(CmdConnectTo, data).Send(teo.auth)
	teo.log.Println("send CmdConnectTo", addr, "(send to teo.auth)")

	return
}

// ConnectToProcess check connection to teonet peer and send answer to client
// and request to Peer (auth server receive connection data from client)
func (teo Teonet) ConnectToProcess(c *Channel, data []byte) (err error) {

	// Unmarshal data
	var con ConnectToData
	con.UnmarshalBinary(data)
	if err != nil {
		teo.log.Println("decode error:", err)
		return
	}
	teo.log.Println("got CmdConnectTo", con.Addr, "from", c)

	// Get channel data and prepare answer data to Client
	ch, ok := teo.Channel(con.Addr)
	if ok {
		// Server IPs and port
		con.IP = ch.c.UDPAddr.IP.String()
		con.Port = uint32(ch.c.UDPAddr.Port)
	} else {
		con.Err = []byte("address not connected")
	}

	// Prepare data and send request to Peer
	if len(con.Err) == 0 {
		var conSer ConnectToData
		// Client address, IPs and port
		conSer.Addr = c.a
		conSer.IP = c.c.UDPAddr.IP.String()
		conSer.Port = uint32(c.c.UDPAddr.Port)
		// Send request to Peer
		_, err = teo.Command(byte(2), data).SendAnswer(ch)
		if err != nil {
			return
		}
	}

	// Send answer to Client
	data, err = con.MarshalBinary()
	if err != nil {
		return
	}
	_, err = teo.Command(CmdConnectTo, data).SendAnswer(c)

	return
}

// connectToAnswerProcess check ConnectTo answer from auth server, connect to
// Peer and send clients teonet addres to it
func (teo Teonet) connectToAnswerProcess(data []byte) (err error) {

	const cantConnectToPeer = "can't connect to peer, error: "

	// Unmarshal data
	var con ConnectToData
	con.UnmarshalBinary(data)
	if err != nil {
		teo.log.Println("unmarshal error:", err)
		return
	}
	teo.log.Println("got CmdConnectTo answer:", con.Addr, con.IP, con.Port,
		string(con.Err))

	// Check server error
	if len(con.Err) != 0 {
		err = errors.New(string(con.Err))
		teo.log.Println(cantConnectToPeer, err)
		return
	}

	// Connect to peer
	c, err := teo.trudp.Connect(con.IP, int(con.Port))
	if err != nil {
		teo.log.Println(cantConnectToPeer, err)
		return
	}

	// Send client teonet address to peer
	c.Send([]byte(teo.config.Address))

	// Add teonet channel
	channel := teo.channels.new(c)
	teo.Connected(channel, con.Addr)

	return
}

// Connected set address to channel and add channel to channels list
func (teo Teonet) Connected(c *Channel, addr string) {
	c.a = addr
	teo.channels.add(c)
}

// ConnectData teonet connect data
type ConnectData struct {
	byteSlice
	PubliKey      []byte // Client public key (generated from private key)
	Address       []byte // Client address (received after connect if empty)
	ServerKey     []byte // Server public key (send if exists or received in connect if empty)
	ServerAddress []byte // Server address (received after connect)
	Err           []byte // Error of connect data processing
}

type byteSlice struct{}

func (b byteSlice) writeSlice(buf *bytes.Buffer, data []byte) (err error) {
	err = binary.Write(buf, binary.LittleEndian, uint16(len(data)))
	if err != nil {
		return
	}
	err = binary.Write(buf, binary.LittleEndian, data)
	return
}

func (b byteSlice) readSlice(buf *bytes.Buffer) (data []byte, err error) {
	var l uint16
	err = binary.Read(buf, binary.LittleEndian, &l)
	if err != nil {
		return
	}
	data = make([]byte, l)
	err = binary.Read(buf, binary.LittleEndian, data)
	return
}

func (c ConnectData) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	c.writeSlice(buf, c.PubliKey)
	c.writeSlice(buf, c.Address)
	c.writeSlice(buf, c.ServerKey)
	c.writeSlice(buf, c.ServerAddress)
	c.writeSlice(buf, c.Err)

	data = buf.Bytes()
	return
}

func (c *ConnectData) UnmarshalBinary(data []byte) (err error) {

	buf := bytes.NewBuffer(data)

	c.PubliKey, err = c.readSlice(buf)
	if err != nil {
		return
	}
	c.Address, err = c.readSlice(buf)
	if err != nil {
		return
	}
	c.ServerKey, err = c.readSlice(buf)
	if err != nil {
		return
	}
	c.ServerAddress, err = c.readSlice(buf)
	if err != nil {
		return
	}
	c.Err, err = c.readSlice(buf)

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

// ConnectToData teonet connect data
type ConnectToData struct {
	byteSlice
	Addr string // Peer address
	IP   string // Peer ip address
	Port uint32 // Peer port
	Err  []byte // Error of connect data processing
}

func (c ConnectToData) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)

	c.writeSlice(buf, []byte(c.Addr))
	c.writeSlice(buf, []byte(c.IP))
	err = binary.Write(buf, binary.LittleEndian, c.Port)
	c.writeSlice(buf, c.Err)

	data = buf.Bytes()
	return
}

func (c *ConnectToData) UnmarshalBinary(data []byte) (err error) {
	buf := bytes.NewBuffer(data)

	d, err := c.readSlice(buf)
	if err != nil {
		return
	}
	c.Addr = string(d)

	d, err = c.readSlice(buf)
	if err != nil {
		return
	}
	c.IP = string(d)

	err = binary.Read(buf, binary.LittleEndian, &c.Port)
	if err != nil {
		return
	}

	c.Err, err = c.readSlice(buf)

	return
}
