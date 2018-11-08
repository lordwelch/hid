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

// A Key is a USB HID value
type Key struct {
	Modifier              []string `json:"modifier"`
	Decimal               byte     `json:"decimal"`
	PressDelayDelimiter   bool     `json:"pressDelayDelimiter,omitempty"`
	ReleaseDelayDelimiter bool     `json:"releaseDelayDelimiter,omitempty"`
	Comment               bool     `json:"comment,omitempty"`
}

// A Keymap is a json representation of the unicode rune mapped to its USB HID value
type Keymap map[string]Key

// Keyboard is a type to attach the methods to if someone wants to use it
type Keyboard struct{}

// bit flag of modifier keys
const (
	LCTRL byte = 1 << iota
	LSHIFT
	LALT
	LSUPER
	RCTRL
	RSHIFT
	RALT
	RSUPER
	NONE = 0
)

var (
	PressDelay      time.Duration // PressDelay is the time in ms to delay before sending a press event
	ReleaseDelay    time.Duration // ReleaseDelay is the time in ms to wait before sending the release event
	KeymapOrder     []string      // Keymap Order is the order in which the specified keymaps cycle on the computer
	KeymapShortcut  [8]byte       // KeymapShortcut is the key combo that will cycle the current keymap by one
	ErrOnUnknownKey bool          // ErrOnUnknownKey whether or not to fail if the unicode rune is invalid or is not in the specified keymaps
	KeymapPath      string        // KeymapPath is the pathe to where the keymap files are stored

	currentKeyMap int
	keys          = make(map[string]Keymap)
	flags         = map[string]byte{
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
					break press
				}
			}

			switch {
			case cur.PressDelayDelimiter:
				var n int
				n, PressDelay = parseDelay(p[index+s:])
				index += s + n
				break press

			case cur.ReleaseDelayDelimiter:
				var n int
				n, ReleaseDelay = parseDelay(p[index+s:])
				index += s + n
				break press

			case cur.Comment:
				var n int
				n = bytes.Index(p[index+s:], []byte("\n"))
				if n < 0 {
					n = 0
				}
				index += s + n
				break press

			default:
				// Calculate next modifier byte
				for _, v := range cur.Modifier {
					mod = mod | flags[v]
				}

				// Set the modifier if it is the first key otherwise
				// check if the next modifier byte is the same
				if i == 2 {
					flag = mod
				} else if flag != mod {
					break press
				}

				// Check for duplicate key press. You can't press a key if it is already pressed.
				for u := 2; u < i; u++ {
					if cur.Decimal == report[u] {
						break press
					}
				}

			}
			report[i] = cur.Decimal
			index += s
			if PressDelay > 0 {
				break press
			}
		}
		report[0] = flag
		r, _ = utf8.DecodeRune(p[index-1:])
		Press(report, Hidg0)
		delay(PressDelay)
	}
	keymapto0() // To make it reproducible
	return index, nil
}

func parseDelay(buffer []byte) (count int, delay time.Duration) {
	var index int
	sb := strings.Builder{}
	for index < len(buffer) {
		r, s := utf8.DecodeRune(buffer[index:])
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
			index += s
		} else {
			if r == '\r' {
				index += s
				r, s = utf8.DecodeRune(buffer[index:])
			}
			if r == '\n' {
				index += s
			}
			break
		}
	}
	i, err := strconv.Atoi(sb.String())
	if err == nil || err == strconv.ErrRange {
		return index, time.Millisecond * time.Duration(i)
	}
	return 0, 0
}

func delay(Delay time.Duration) {
	if Delay > 0 {
		if syncCheck, ok := Hidg0.(syncer); ok {
			syncCheck.Sync()
		}
		time.Sleep(Delay)
	}
}

func Press(press [8]byte, file io.Writer) {
	file.Write(press[:])
	delay(ReleaseDelay)
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
		_, ok := CurrentKeymap()[string(r)]
		if ok {
			Hidg0.Write(buf.Bytes())
			return true
		}
		Press(KeymapShortcut, buf)
		if currentKeyMap == len(KeymapOrder)-1 {
			currentKeyMap = 0
		} else {
			currentKeyMap++
		}

	}
	return false
}

func CurrentKeymap() Keymap {
	keymap, ok := keys[KeymapOrder[currentKeyMap]]
	if ok {
		return keymap
	}
	return LoadKeymap(KeymapOrder[currentKeyMap])

}

func LoadKeymap(keymapName string) Keymap {
	var (
		err     error
		content []byte
		file    = path.Join(path.Join(KeymapPath, "hid"), keymapName+".json")
		tmp     = make(Keymap, 0)
	)
	fmt.Println(file)
	content, err = ioutil.ReadFile(file)
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
