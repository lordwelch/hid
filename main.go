package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	hid "gitea.narnian.us/lordwelch/hid/ghid"
)

func contains(str string, find []string) bool {
	str = strings.ToLower(str)
	for _, s := range find {
		if strings.Contains(str, strings.ToLower(s)) {
			return true
		}
	}
	return false
}

func parse_shortcut(shortcut string) ([8]byte, error) {
	var (
		modifiers        = []string{}
		key              = ""
		curModifier byte = 0
		curKey      byte = 0
	)
	strs := strings.SplitN(strings.ToLower(shortcut), " ", 2)
	if len(strs) > 1 {
		modifiers = strings.Split(strs[0], "|")
		key = strings.TrimSpace(strs[1])
	} else {
		if contains(strs[0], hid.AllModifiers) {
			modifiers = strings.Split(strs[0], "|")
		} else {
			key = strings.TrimSpace(strs[0])
		}
	}
	for _, v := range modifiers {
		curModifier |= hid.Modifiers[strings.TrimSpace(v)]
	}
	if id, ok := hid.StandardKeys[key]; ok {
		curKey = id
	} else {
		return [8]byte{}, fmt.Errorf("Key %q not found", key)
	}
	return [8]byte{curModifier, 0x0, curKey}, nil
}

func main() {
	var (
		Shortcut     string
		filePath     string
		keymapPath   string
		ghidPath     string
		pressDelay   time.Duration
		releaseDelay time.Duration
		err          error
		ghid         *os.File
		tmp          *os.File
		keyboard     *hid.Keyboard
		keymaps      []string
	)
	if _, exists := os.LookupEnv("XDG_CONFIG_HOME"); !exists {
		_ = os.Setenv("XDG_CONFIG_HOME", path.Join(os.ExpandEnv("$HOME"), ".config"))
	}
	flag.StringVar(&Shortcut, "shortcut", "", "Keymap cycle shortcut")
	flag.StringVar(&Shortcut, "s", "LALT ⇪", "Keymap cycle shortcut")
	flag.StringVar(&keymapPath, "path", path.Join(os.ExpandEnv("$XDG_CONFIG_HOME"), "hid"), "Path to config dir default: $XDG_CONFIG_HOME")
	flag.StringVar(&keymapPath, "p", path.Join(os.ExpandEnv("$XDG_CONFIG_HOME"), "hid"), "Path to config dir default: $XDG_CONFIG_HOME")
	flag.StringVar(&filePath, "f", "-", "The file to read content from. Defaults to stdin")
	flag.StringVar(&filePath, "file", "-", "The file to read content from. Defaults to stdin")
	flag.StringVar(&ghidPath, "g", "/dev/hidg0", "The device to send key presses to. Defaults to /dev/hidg0")
	flag.StringVar(&ghidPath, "ghid", "/dev/hidg0", "The device to send key presses to. Defaults to /dev/hidg0")
	flag.DurationVar(&pressDelay, "press", 0, "sets the default delay between presses of individual keys")
	flag.DurationVar(&releaseDelay, "release", 0, "sets the default delay between sending the press of an individual key and sending the release")
	flag.Parse()
	if flag.NArg() < 0 {
		flag.Usage()
		os.Exit(1)
	}
	fmt.Println(keymapPath)
	dirs, err := os.ReadDir(keymapPath)
	if err != nil {
		panic(err)
	}

keymap:
	for _, requestedKeymap := range flag.Args() {
		for _, dir := range dirs {
			if strings.HasPrefix(strings.ToLower(dir.Name()), strings.ToLower(requestedKeymap)) {
				keymaps = append(keymaps, strings.TrimSuffix(dir.Name(), ".json"))
				break keymap
			}
		}
		panic(fmt.Sprintf("Keymap %q not found", requestedKeymap))
	}

	if filePath != "-" {
		tmp, err = os.Open(path.Clean(filePath))
		if err == nil {
			_ = os.Stdin.Close()
			os.Stdin = tmp
		}
	}

	ghid, err = os.OpenFile(ghidPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer ghid.Close()

	keyboard = hid.NewKeyboard(hid.Modifiers, keymaps, keymapPath, ghid)
	keyboard.PressDelay = pressDelay
	keyboard.ReleaseDelay = releaseDelay
	keyboard.KeymapShortcut, err = parse_shortcut(Shortcut)
	if err != nil {
		panic(fmt.Errorf("error parsing shortcut: %w", err))
	}

	_, err = io.Copy(keyboard, os.Stdin)

	if err != nil {
		panic(err)
	}

	fmt.Println("Success!")
}
