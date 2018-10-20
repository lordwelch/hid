package main

import (
	"flag"
	"fmt"
	"os"
	"timmy.narnian.us/hid/ghid"
)

func main() {
	var (
		SHORTCUT string
	)

	flag.StringVar(&SHORTCUT, "shortcut", "", "Keymap cycle shortcut")
	flag.StringVar(&SHORTCUT, "s", "", "Keymap cycle shortcut")
	flag.StringVar(&hid.KeymapPath, "path", os.ExpandEnv("$XDG_CONFIG_HOME"), "Path to config dir default: $XDG_CONFIG_HOME")
	flag.StringVar(&hid.KeymapPath, "p", os.ExpandEnv("$XDG_CONFIG_HOME"), "Path to config dir default: $XDG_CONFIG_HOME")
	flag.Parse()

	hid.KeymapOrder = flag.Args()

	fmt.Println(hid.KeymapPath)

	file, err := os.OpenFile("/dev/hidg0", os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(err)
	}
	hid.Hidg0 = file
	defer file.Close()

	hid.Write(os.Stdin)

	if err != nil {
		panic(err)
	}

	fmt.Println("Success!")
}
