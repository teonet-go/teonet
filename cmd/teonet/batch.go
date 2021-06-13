package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kirill-scherba/teonet"
	"github.com/kirill-scherba/teonet/cmd/teonet/menu"
)

const (
	aliasBatchFile   = "alias.conf"
	connectBatchFile = "connectto.conf"
)

type Batch struct{ menu *menu.Menu }

// type AliasData struct {
// 	alias   string
// 	address string
// }

// run aliases from config file
func (b *Batch) run(name string) (err error) {
	// Get file name
	fname, err := b.file(name)
	if err != nil {
		return
	}

	// Open file
	f, err := os.Open(fname)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Printf("\nrun batch: %s\n\n", fname)
	space := regexp.MustCompile(`\s+`)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		line = space.ReplaceAllString(line, " ")

		fmt.Println(line) // Println will add back the final '\n'
		if err = b.menu.ExecuteCommand(line); err != nil {
			fmt.Println("error:", err)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
	return
}

// write write aliase to config file
// func (b Batch) write() {
// }

func (b Batch) file(name string) (f string, err error) {
	f, err = os.UserConfigDir()
	if err != nil {
		return
	}
	f += "/" + teonet.ConfigDir + "/" + appShort + "/" + name
	return
}

// func GetAlias(m *menu.Menu) {
// 	b := new(Batch)
// 	b.read(m, aliasConfigFile)
// }