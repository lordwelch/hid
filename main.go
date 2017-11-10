package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/alexflint/go-arg"
)

type Key struct {
	Modifier string `json:"modifier"`
	Decimal  int    `json:"decimal"`
}

type Keys map[string]Key

type Args struct {
	SHORTCUT string   `arg:"-S,help:Keymap cycle shortcut"`
	ORDER    []string `arg:"positional,help:Order of keymaps"`
}

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

	fmt.Println([8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
}

func Hold(press [8]byte, file io.Writer) {
	binary.Write(file, binary.BigEndian, press[:])
}

func changeKeymap(r rune, keys map[string]Keys, args Args, hidg0 *os.File, currentKeyMap *int) {
	//fmt.Println(*currentKeyMap)
	//fmt.Println(args)
	kmap := args.ORDER[(*currentKeyMap)]

	for keys[kmap][string(r)].Decimal == 0 {
		Press([8]byte{LCTRL, 0x00, 0x57, 0x00, 0x00, 0x00, 0x00, 0x00}, hidg0)
		*currentKeyMap++
		if *currentKeyMap == len(keys) {
			panic("key not in keymap")
		}
	}
}

func main() {
	var (
		args          Args
		envExists     bool
		env           string
		hidg0         *os.File
		err           error
		keymapsF      []os.FileInfo
		keys          = make(map[string]Keys)
		cfgPath       string
		stdin         = bufio.NewReader(os.Stdin)
		currentKeyMap int
		flags         = map[string]byte{
			"LSHIFT": LSHIFT,
			"LCTRL":  LCTRL,
			"LALT":   LALT,
			"LSUPER": LSUPER,
			"RSHIFT": RSHIFT,
			"RCTRL":  RCTRL,
			"RALT":   RALT,
			"RSUPER": RSUPER,
			"NONE":   0,
		}
	)
	arg.MustParse(&args)
	env, envExists = os.LookupEnv("XDG_CONFIG_HOME")
	if !envExists {
		env = os.Getenv("HOME")
	}

	cfgPath = path.Join(env, "hid")
	keymapsF, err = ioutil.ReadDir(cfgPath)
	if err != nil {
		panic(err)
	}

	fmt.Println(cfgPath)

	hidg0, err = os.OpenFile("/dev/hidg0", os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}

	for _, file := range keymapsF {
		var (
			ext string
		)

		ext = path.Ext(file.Name())

		if strings.ToLower(ext) == ".json" {
			var (
				tmp     Keys
				T       *os.File
				content []byte
			)

			T, err = os.Open(path.Join(cfgPath, file.Name()))
			if err != nil {
				panic(err)
			}

			content, err = ioutil.ReadAll(T)
			if err != nil {
				panic(err)
			}

			err = json.Unmarshal(content, &tmp)
			if err != nil {
				panic(err)
			}

			//fmt.Println(strings.TrimSuffix(file.Name(), ext))
			keys[strings.TrimSuffix(file.Name(), ext)] = tmp
			T.Close()
		}
	}
	fmt.Println(keys)
	for {
		var (
			r      rune
			flag   byte
			report [6]byte
		)

		r, _, err = stdin.ReadRune()
		//fmt.Printf("%s\n", string(r))

		if err == io.EOF {
			break
		}

		if err != nil {
			panic(err)
		}
		changeKeymap(r, keys, args, hidg0, &currentKeyMap)
		flag = flags[keys[args.ORDER[currentKeyMap]][string(r)].Modifier]
		binary.BigEndian.PutUint16(report[:], uint16(keys[args.ORDER[currentKeyMap]][string(r)].Decimal))
		Press([8]byte{flag, 0, report[0], report[1], report[2], report[3], report[4], report[5]}, hidg0)

	}
	fmt.Println("Success!")
	hidg0.Close()

}
