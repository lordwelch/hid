package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	hid "github.com/lordwelch/hid/ghid"
)

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
	)
	if _, exists := os.LookupEnv("XDG_CONFIG_HOME"); !exists {
		_ = os.Setenv("XDG_CONFIG_HOME", path.Join(os.ExpandEnv("$HOME"), ".config"))
	}
	flag.StringVar(&Shortcut, "shortcut", "", "Keymap cycle shortcut")
	flag.StringVar(&Shortcut, "s", "", "Keymap cycle shortcut")
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

	keyboard = hid.NewKeyboard(hid.Modifiers, flag.Args(), keymapPath, ghid)
	keyboard.PressDelay = pressDelay
	keyboard.ReleaseDelay = releaseDelay

	_, err = io.Copy(keyboard, os.Stdin)

	if err != nil {
		panic(err)
	}

	fmt.Println("Success!")
}
