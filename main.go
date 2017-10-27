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
		test    [8]byte = [...]byte{0x00, 0x00, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
		unpress [8]byte = [...]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		write   []byte
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
	//test[0] |= LCTRL
	fmt.Printf("%08b\n", test[0])
	file, err := os.OpenFile("/dev/hidg0", os.O_WRONLY, os.ModePerm)

	fmt.Println(err)
	for i := 1; i <= 100; i++ {
		write = append(write, test[:]...)
		write = append(write, unpress[:]...)
	}
	binary.Write(file, binary.BigEndian, write)

	file.Close()

}
