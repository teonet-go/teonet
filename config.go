package teonet

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/kirill-scherba/tru"
)

var nMODULEconf = "Config"

func (teo *Teonet) newConfig(appName string, osConfigDir string) (err error) {
	// New config holder
	teo.config = &config{appName: appName, osConfigDir: osConfigDir}

	// Check config file exists and create and save new config if does not exists
	err = teo.config.exists()
	if err != nil {
		log.Error.Println(nMODULEconf, err)
		err = teo.config.create()
		if err != nil {
			return
		}
		log.Error.Println(nMODULEconf, "new keys and config file created")
	}

	// Read config file
	err = teo.config.read()
	if err != nil {
		return
	}

	return
}

// config teonet
type config struct {
	TrudpPrivateKeyData []byte          `json:"trudp_private_key"`
	PrivateKeyData      []byte          `json:"private_key"`
	ServerPublicKeyData []byte          `json:"server_key"`
	Address             string          `json:"address"`
	trudpPrivateKey     *rsa.PrivateKey `json:"-"`
	appName             string          `json:"-"`
	osConfigDir         string          `json:"-"`
}

type OsConfigDir string

const (
	ConfigDir        = "teonet"
	configFile       = "teonet.conf"
	configBufferSize = 2048
)

func (c config) marshal() (data []byte, err error) {
	data, err = json.MarshalIndent(c, "", " ")
	if err != nil {
		return
	}
	return
}

func (c *config) unmarshal(data []byte) error {
	return json.Unmarshal(data, c)
}

// file get file name
func (c config) file() (res string, err error) {
	return c.configFile(c.appName, configFile)
}

// configFile return config file full name (path + name)
// TODO: if os.UserConfigDir() return err - do something right
func (c config) configFile(appName string, file string) (res string, err error) {
	if c.osConfigDir != "" {
		res = c.osConfigDir
	} else {
		res, err = os.UserConfigDir()
		if err != nil {
			return
		}
	}
	res += "/" + ConfigDir + "/" + appName + "/" + file
	return
}

// ConfigFile return full path to config file
func (teo Teonet) ConfigFile(appName string, file string) (res string, err error) {
	return teo.config.configFile(appName, file)
}

func (c config) save() (err error) {

	file, err := c.file()
	if err != nil {
		return
	}

	f, err := os.Create(file)
	if err != nil {
		return
	}

	data, err := c.marshal()
	if err != nil {
		return
	}

	_, err = f.Write(data)

	return

	// var prettyJSON bytes.Buffer
	// error := json.Indent(&prettyJSON, body, "", "\t")
	// if error != nil {
	//     log.Println("JSON parse error: ", error)
	//     App.BadRequest(w)
	//     return
	// }
}

// exists return nil if config file exists
func (c config) exists() (err error) {
	file, err := c.file()
	if err != nil {
		return
	}

	_, err = os.Stat(file)
	return
}

// read config file and parse private keys
func (c *config) read() (err error) {

	// Get file name
	file, err := c.file()
	if err != nil {
		return
	}

	// Open file
	f, err := os.Open(file)
	if err != nil {
		return
	}

	// Read file data
	data := make([]byte, configBufferSize)
	n, err := f.Read(data)
	if err != nil {
		return
	}
	if n == configBufferSize {
		err = errors.New("too small read buffer")
		return
	}

	// Unmarshal config data
	err = c.unmarshal(data[:n])
	if err != nil {
		return
	}

	// Parse trudp private key
	c.trudpPrivateKey, err = x509.ParsePKCS1PrivateKey(c.TrudpPrivateKeyData)
	if err != nil {
		return
	}

	// Get teonet address
	c.Address, err = c.makeAddress(c.PrivateKeyData)
	if err != nil {
		return
	}
	// c.log.Printf("teonet address: %s\n", c.Address)

	return
}

// create new config with new private keys and save it to config folder
func (c *config) create() (err error) {

	file, err := c.file()
	if err != nil {
		return
	}

	err = os.MkdirAll(path.Dir(file), os.ModePerm)
	if err != nil {
		return
	}

	// Create new config holder
	*c = config{appName: c.appName, osConfigDir: c.osConfigDir}
	err = c.createKeys()
	if err != nil {
		return
	}

	return c.save()
}

// createKeys create new trudp and teonet private keys
func (c *config) createKeys() (err error) {

	// Create trudp rsa private key
	c.trudpPrivateKey, err = tru.GeneratePrivateKey()
	if err != nil {
		return
	}
	c.TrudpPrivateKeyData = x509.MarshalPKCS1PrivateKey(c.trudpPrivateKey)

	// Create teonet (address) private key
	c.PrivateKeyData = c.generatePrivateKey()
	fmt.Printf("new private key hex: %x\n", c.PrivateKeyData)

	return
}

// generatePrivateKey create new teonet private key
func (c config) generatePrivateKey() (key []byte) {
	buf := make([]byte, 512)
	io.ReadFull(rand.Reader, buf)
	h := sha256.New()
	h.Write(buf)
	key = h.Sum(nil)
	return
}

// getPublicKey get teonet public key from private key
func (c config) getPublicKey() (key []byte) {
	h := sha256.New()
	h.Write(c.PrivateKeyData)
	key = h.Sum(nil)
	return
}

// GetPrivateKey get teonet private key
func (teo Teonet) GetPrivateKey() (key []byte) {
	return teo.config.PrivateKeyData
}

// GetPublicKey get teonet public key from private key
func (t Teonet) GetPublicKey() []byte {
	return t.config.getPublicKey()
}

// makeAddress get teonet address from private key
func (c config) makeAddress(keyData []byte) (addr string, err error) {
	const addrLen = 35
	var escaper = strings.NewReplacer("+", "", "/", "", "=", "")
	addr = base64.StdEncoding.EncodeToString(keyData)
	addr = escaper.Replace(addr)
	if len(addr) < addrLen {
		err = errors.New("too low address len")
		return
	}
	addr = addr[:addrLen]
	return
}

// Address get teonet address
func (t Teonet) Address() (addr string) {
	return t.config.Address
}

// MakeAddress make teonet address from key
func (t Teonet) MakeAddress(keyData []byte) (addr string, err error) {
	return t.config.makeAddress(keyData)
}
