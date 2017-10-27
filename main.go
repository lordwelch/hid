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
		test    [8]byte = [8]byte{0x00, 0x00, 0x52, 0x00, 0x00, 0x00, 0x00, 0x00}
		test1   [8]byte = [8]byte{0x00, 0x00, 0x4c, 0x00, 0x00, 0x00, 0x00, 0x00}
		unpress [8]byte = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
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
	for i := 1; i <= 1000; i++ {
		write = append(write, test[:]...)
		write = append(write, unpress[:]...)
		write = append(write, test1[:]...)
		write = append(write, unpress[:]...)
	}
	binary.Write(file, binary.BigEndian, write)

	file.Close()

}
