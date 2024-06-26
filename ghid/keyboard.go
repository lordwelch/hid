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

type Keyboard struct {
	PressDelay      time.Duration // PressDelay is the time in ms to delay before sending a press event
	ReleaseDelay    time.Duration // ReleaseDelay is the time in ms to wait before sending the release event
	KeymapOrder     []string      // Keymap Order is the order in which the specified keymaps cycle on the computer
	KeymapShortcut  [8]byte       // KeymapShortcut is the key combo that will cycle the current keymap by one
	ErrOnUnknownKey bool          // ErrOnUnknownKey whether or not to fail if the unicode rune is invalid or is not in the specified keymaps
	KeymapPath      string        // KeymapPath is the pathe to where the keymap files are stored
	currentKeyMap   int
	keymaps         map[string]Keymap
	flags           map[string]byte
	Hidg0           io.Writer
}

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
	Modifiers = map[string]byte{
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
	AllModifiers = []string{
		"LSHIFT",
		"LCTRL",
		"LALT",
		"LSUPER",
		"RSHIFT",
		"RCTRL",
		"RALT",
		"RSUPER",
		"NONE",
	}
)

func NewKeyboard(Modifiers map[string]byte, kemapOrder []string, KeymapPath string, hidg0 io.Writer) *Keyboard {
	return &Keyboard{
		flags:       Modifiers,
		KeymapOrder: kemapOrder,
		KeymapPath:  KeymapPath,
		Hidg0:       hidg0,
	}
}

// io.writer probably isn't the best interface to use for this
func (k *Keyboard) Write(p []byte) (int, error) {
	var (
		index int
		err   error
	)
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
			cur, ok := k.CurrentKeymap()[string(r)]
			if !ok {
				if i == 2 { // We can change the keymap if we are on the first key
					ok, err = k.changeKeymap(r)
					if !ok { // rune does not have a mapping
						if k.ErrOnUnknownKey {
							if err != nil {
								return index, err
							}
							return index, fmt.Errorf("rune not in keymap: %c", r)
						}
						index += s
						break press
					}
				} else { // rune does not have a mapping in this keymaps
					break press
				}
			}

			switch {
			case cur.PressDelayDelimiter:
				var n int
				n, k.PressDelay = parseDelay(p[index+s:])
				index += s + n
				break press

			case cur.ReleaseDelayDelimiter:
				var n int
				n, k.ReleaseDelay = parseDelay(p[index+s:])
				index += s + n
				break press

			case cur.Comment:
				var n int
				n = bytes.Index(p[index+s:], []byte("\n")) + 1
				if n < 0 {
					n = 0
				}
				index += s + n
				break press
			case r == '␀':
				// Causes immediate key press useful for modifier keys
				index += s
				break press

			default:
				// Calculate next modifier byte
				for _, v := range cur.Modifier {
					mod |= k.flags[v]
				}

				// Set the modifier if it is the first key otherwise
				// check if the next modifier byte is the same
				if i == 2 {
					flag = mod
				} else if flag != mod {
					// This is the second key press if the previous one was a modifier only Decimal == 0 then take the current key as well
					if report[i-1] != 0 {
						break press
					}
					// Add the modifier of the current key eg 'D' adds shift; 'd' does not
					flag |= mod
				}

				// Check for duplicate key press. You can't press a key if it is already pressed, unless it is 0 indicating a modifier.
				for u := 2; u < i; u++ {
					if cur.Decimal == report[u] && cur.Decimal != 0 {
						break press
					}
				}
			}
			report[i] = cur.Decimal
			index += s
			if k.PressDelay > 0 {
				// This is the first key press if this is a modifier only Decimal == 0 then take the next key as well
				if report[i] != 0 {
					break press
				}
			}
		}
		report[0] = flag
		err = k.Press(report, k.Hidg0)
		if err != nil {
			return index, err
		}
		k.delay(k.PressDelay)
	}
	err = k.keymapto0() // To make it reproducible
	if err != nil {
		return index, err
	}
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

func (k *Keyboard) delay(Delay time.Duration) {
	if Delay > 0 {
		if syncCheck, ok := k.Hidg0.(syncer); ok {
			_ = syncCheck.Sync()
		}
		time.Sleep(Delay)
	}
}

func (k *Keyboard) Press(press [8]byte, file io.Writer) error {
	_, err1 := file.Write(press[:])
	k.delay(k.ReleaseDelay)
	_, err2 := file.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func (k *Keyboard) Hold(press [8]byte, file io.Writer) error {
	_, err := file.Write(press[:])
	return err
}

func (k *Keyboard) keymapto0() error {
	if len(k.KeymapOrder) > 1 {
		for i := 0; i < len(k.KeymapOrder)-(k.currentKeyMap); i++ {
			err := k.Press(k.KeymapShortcut, k.Hidg0)
			if err != nil {
				return err
			}
		}
		k.currentKeyMap = 0
	}
	return nil
}

func (k *Keyboard) changeKeymap(r rune) (bool, error) {
	var err error
	buf := bytes.NewBuffer(make([]byte, 0, 8*len(k.KeymapOrder))) // To batch shortcut presses

	for i := 0; i < len(k.KeymapOrder); i++ {
		_, ok := k.CurrentKeymap()[string(r)]
		if ok {
			_, err = k.Hidg0.Write(buf.Bytes())
			return true, err
		}
		err = k.Press(k.KeymapShortcut, buf)
		if err != nil {
			return false, err
		}
		if k.currentKeyMap == len(k.KeymapOrder)-1 {
			k.currentKeyMap = 0
		} else {
			k.currentKeyMap++
		}
	}
	return false, nil
}

func (k *Keyboard) CurrentKeymap() Keymap {
	keymap, ok := k.keymaps[k.KeymapOrder[k.currentKeyMap]]
	if ok {
		return keymap
	}
	if k.keymaps == nil {
		k.keymaps = make(map[string]Keymap)
	}
	k.keymaps[k.KeymapOrder[k.currentKeyMap]] = LoadKeymap(k.KeymapOrder[k.currentKeyMap], k.KeymapPath)
	return k.keymaps[k.KeymapOrder[k.currentKeyMap]]
}

func LoadKeymap(keymapName string, KeymapPath string) Keymap {
	var (
		err     error
		content []byte
		file    = path.Join(KeymapPath, keymapName+".json")
		tmp     = make(Keymap)
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
	return tmp
}
