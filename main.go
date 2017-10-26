package main

import (
	"fmt"
	"os"
)

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

func main() {
	var (
		test [8]byte = [8]byte{0x00, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a}
	)
	fmt.Printf("%08b\n%08b\n%08b\n%08b\n%08b\n%08b\n%08b\n%08b\n\n", LCTRL,
		LSHIFT,
		LALT,
		LSUPER,
		RCTRL,
		RSHIFT,
		RALT,
		RSUPER)
	fmt.Println()
	fmt.Printf("%08b\n", test[0])
	test[0] |= LCTRL
	fmt.Printf("%08b\n", test[0])
	file, _ := os.Open("/dev/hidg0")
	file.Write(test[:])

}
