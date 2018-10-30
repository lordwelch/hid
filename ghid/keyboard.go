package hid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type syncer interface {
	Sync() error
}

type Key struct {
	Modifier       []string `json:"modifier"`
	Decimal        byte     `json:"decimal"`
	DelayDelimiter bool     `json:"delayDelimiter,omitempty"`
}

type Keymap map[string]Key

type Keyboard struct{}

const NONE = 0
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

const delayDelimiter = "⏲"

var (
	currentKeyMap int
	DefaultDelay  time.Duration

	KeymapOrder     []string = []string{"qwerty"}
	KeymapShortcut  [8]byte
	ErrOnUnknownKey bool
	KeymapPath      string
	keys            = make(map[string]Keymap)
	flags           = map[string]byte{
		"LSHIFT": LSHIFT,
		"LCTRL":  LCTRL,
		"LALT":   LALT,
		"LSUPER": LSUPER,
		"RSHIFT": RSHIFT,
		"RCTRL":  RCTRL,
		"RALT":   RALT,
		"RSUPER": RSUPER,
		"NONE":   NONE,
	}
	Hidg0 io.Writer
)

func (k Keyboard) Write(p []byte) (n int, err error) {
	return write(p)
}

func Write(r io.Reader) error {
	_, err := io.Copy(Keyboard{}, r)
	return err
}

// io.writer probably isn't the best interface to use for this
func write(p []byte) (n int, err error) {
	var index int
	for index < len(p) {
		var (
			r      rune
			s      int
			flag   byte
			report [8]byte
		)
	press:
		for i := 2; i < 8 && index < len(p); i++ {
			var (
				mod byte
			)
			if DefaultDelay > 0 && i > 2 {
				break
			}
			r, s = utf8.DecodeRune(p[index:])
			if r == utf8.RuneError {
				return index, fmt.Errorf("invalid rune: 0x%X", p[index]) // This probably screws things up if the last rune in 'p' is incomplete
			}
			cur, ok := CurrentKeymap()[string(r)]
			if !ok {
				if i == 2 { // can't press two keys from different keymaps
					if !changeKeymap(r) && ErrOnUnknownKey {
						return index, fmt.Errorf("rune not in keymap: %c", r)
					}
				} else {
					break
				}
			}

			// Check if this is a delay
			if cur.DelayDelimiter {
				index += s + parseDelay(p[index+s:])
				break
			}

			// Calculate next modifier byte
			for _, v := range cur.Modifier {
				mod = mod | flags[v]
			}
			// Set the modifier if it is the first key otherwise
			// check if the next modifier byte is the same
			if i == 2 {
				flag = mod
			} else if flag != mod {
				break
			}

			// Check for duplicate key press. You can't press a key if it is already pressed.
			for u := 2; u < i; u++ {
				if cur.Decimal == report[u] {
					break press
				}
			}
			report[i] = cur.Decimal
			index += s
		}
		report[0] = flag
		r, _ = utf8.DecodeRune(p[index-1:])
		fmt.Printf("%c: delay: %v %v\n", r, DefaultDelay, DefaultDelay > (0))
		Press(report, Hidg0)
		delay()
	}
	keymapto0() // To make it reproducible
	return index, nil
}

func parseDelay(buffer []byte) int {
	var index int
	sb := strings.Builder{}
	for index < len(buffer) {
		r, s := utf8.DecodeRune(buffer[index:])
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
			index += s
		} else {
			break
		}
	}
	i, err := strconv.Atoi(sb.String())
	if err == nil || err == strconv.ErrRange {
		DefaultDelay = time.Millisecond * time.Duration(i)
		return index
	}
	return 0
}

func delay() {
	if DefaultDelay > 0 {
		if syncCheck, ok := Hidg0.(syncer); ok {
			syncCheck.Sync()
		}
		time.Sleep(DefaultDelay)
		// DefaultDelay = 0
	}
}

func Press(press [8]byte, file io.Writer) {
	file.Write(press[:])
	Hidg0.(syncer).Sync()
	time.Sleep(DefaultDelay)
	file.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
}

func Hold(press [8]byte, file io.Writer) {
	file.Write(press[:])
}

func keymapto0() {
	if len(KeymapOrder) > 1 {
		for i := 0; i < len(KeymapOrder)-(currentKeyMap); i++ {
			Press([8]byte{LALT, 0x00, 0x39, 0x00, 0x00, 0x00, 0x00, 0x00}, Hidg0)
		}
		currentKeyMap = 0
	}
}

func changeKeymap(r rune) bool {
	buf := bytes.NewBuffer(make([]byte, 0, 8*len(KeymapOrder))) // To batch shortcut presses

	for i := 0; i < len(KeymapOrder); i++ {
		if _, ok := CurrentKeymap()[string(r)]; ok {
			Hidg0.Write(buf.Bytes())
			return true
		} else {
			Press(KeymapShortcut, buf)
			if currentKeyMap == len(KeymapOrder)-1 {
				currentKeyMap = 0
			} else {
				currentKeyMap++
			}
		}
	}
	return false
}

func CurrentKeymap() Keymap {
	if keymap, ok := keys[KeymapOrder[currentKeyMap]]; ok {
		return keymap
	} else {
		return LoadKeymap(KeymapOrder[currentKeyMap])
	}
}

func LoadKeymap(keymapName string) Keymap {
	var (
		err     error
		content []byte
		tmp     = make(Keymap, 0)
	)
	fmt.Println(path.Join(path.Join(KeymapPath, "hid"), keymapName+".json"))
	content, err = ioutil.ReadFile(path.Join(path.Join(KeymapPath, "hid"), keymapName+".json"))
	if err != nil {
		return nil
	}

	err = json.Unmarshal(content, &tmp)
	if err != nil {
		return nil
	}

	//fmt.Println(strings.TrimSuffix(file.Name(), ext))
	keys[keymapName] = tmp
	return keys[keymapName]
}
