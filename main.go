package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/alexflint/go-arg"
)

type Key struct {
	name     rune
	modifier string
	decimal  int
}

type Keys []Key

const (
	LCTRL byte = 1 << iota
	LSHIFT
	LALT
	LSUPER
	RCTRL
	RSHIFT
	RALT
	RSUPER
)

func Press(press [8]byte, file io.Writer) {
	binary.Write(file, binary.BigEndian, press[:])
	binary.Write(file, binary.BigEndian, [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
}

func Hold(press [8]byte, file io.Writer) {
	binary.Write(file, binary.BigEndian, press[:])
}

func main() {
	var (
		args struct {
			SHORTCUT string   `arg:"-S,help:Keymap cycle shortcut"`
			ORDER    []string `arg:positional,help:Order of keymaps`
		}
		hidg0         *os.File
		err           error
		keymapsF      []os.FileInfo
		keys          map[string]Keys
		cfgPath       = path.Join(os.Getenv("XDG_CONFIG_HOME"), "hid")
		stdin         = bufio.NewReader(os.Stdin)
		currentKeyMap int
		good          bool
	)
	arg.MustParse(&args)
	keymapsF, err = ioutil.ReadDir(cfgPath)
	if err != nil {
		panic(err)
	}

	hidg0, err = os.OpenFile("/dev/hidg0", os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}

	for _, file := range keymapsF {
		var (
			ext string
		)

		ext = path.Ext(file.Name())
		if strings.ToLower(ext) == "json" {
			var (
				tmp     Keys
				T       *os.File
				content []byte
			)
			T, err = os.Open(strings.TrimSuffix(file.Name(), ext))
			if err != nil {
				panic(err)
			}

			content, err = ioutil.ReadAll(T)
			if err != nil {
				panic(err)
			}

			json.Unmarshal(content, tmp)
			keys[strings.TrimSuffix(file.Name(), ext)] = tmp
			T.Close()
		}
	}
	for good {
		var r rune
		r, _, err = stdin.ReadRune()
		for keys[args.ORDER[currentKeyMap]][r].name != r {
			Press([8]byte{LCTRL, 0x00, 0x57, 0x00, 0x00, 0x00, 0x00, 0x00}, hidg0)
			currentKeyMap++
		}

	}

	hidg0.Close()

}
