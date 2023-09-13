package menu

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/teonet-go/teonet"
)

type Batch struct{ menu *Menu }

// NewBatch creates new Batch object
func NewBatch(menu *Menu) *Batch { return &Batch{menu} }

// Run executes commands from batch config file
func (b *Batch) Run(appShort, name string) (err error) {
	// Get file name
	fname, err := b.file(appShort, name)
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
		if len(line) == 0 {
			continue
		}
		if err = b.menu.ExecuteCommand(line); err != nil {
			fmt.Println("error:", err)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
	return
}

// Save batch to config file
func (b Batch) Save(appShort, name string, prefix string, batch []string) (err error) {
	// Get file name
	fname, err := b.file(appShort, name)
	if err != nil {
		return
	}

	// Create or open file for write
	f, err := os.Create(fname)
	if err != nil {
		return
	}
	defer f.Close()

	// Write batch to file
	datawriter := bufio.NewWriter(f)
	for _, data := range batch {
		_, _ = datawriter.WriteString(prefix + " " + data + "\n")
	}
	datawriter.Flush()

	return
}

// file return full file name with config dir folder
func (b Batch) file(appShort, name string) (f string, err error) {
	f, err = os.UserConfigDir()
	if err != nil {
		return
	}
	f += "/" + teonet.ConfigDir + "/" + appShort + "/" + name
	return
}
