package main

import (
	"encoding/binary"
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
		test [16]byte = [...]byte{0x00, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
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
	fmt.Printf("%08b\n", test[:])
	test[0] |= LCTRL
	fmt.Printf("%08b\n", test[:])
	file, err := os.OpenFile("/dev/hidg0", os.O_WRONLY, os.ModePerm)
	file2, err2 := os.OpenFile("test", os.O_WRONLY|os.O_CREATE, os.ModePerm)
	fmt.Println(err)
	fmt.Println(err2)
	binary.Write(file, binary.BigEndian, test[:])
	binary.Write(file2, binary.BigEndian, test[:])
	file.Close()
	file2.Close()
}
