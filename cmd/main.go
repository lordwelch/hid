package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"timmy.narnian.us/git/timmy/hid"
)

func main() {
	var (
		err error

		args struct {
			SHORTCUT string   `arg:"-S,help:Keymap cycle shortcut"`
			PATH     string   `arg:"-P,help:Path to config dir default: $XDG_CONFIG_HOME"`
			ORDER    []string `arg:"required,positional,help:Order of keymaps"`
		}
	)
	args.PATH = os.ExpandEnv("$XDG_CONFIG_HOME")
	arg.MustParse(&args)
	hid.KeymapOrder = args.ORDER

	hid.KeymapPath = args.PATH
	fmt.Println(hid.KeymapPath)

	hid.Hidg0, err = os.OpenFile("hidg0", os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(err)
	}
	hid.Write(transform.NewReader(os.Stdin, unicode.BOMOverride(unicode.UTF8.NewDecoder())))

	// if err != nil {
	// 	panic(err)
	// }

	fmt.Println("Success!")
}
