package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	hid "timmy.narnian.us/hid/ghid"
)

func main() {
	var (
		Shortcut   string
		filePath   string
		keymapPath string
		err        error
		ghid0      *os.File
		tmp        *os.File
		keyboard   *hid.Keyboard
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

	ghid0, err = os.OpenFile("/dev/hidg0", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer ghid0.Close()

	keyboard = hid.NewKeyboard(hid.Modifiers, flag.Args(), keymapPath, ghid0)

	_, err = io.Copy(keyboard, os.Stdin)

	if err != nil {
		panic(err)
	}

	fmt.Println("Success!")
}
