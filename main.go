package main

import (
	"encoding/binary"
	"fmt"
	"io"
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

func Press(press [8]byte, file io.Writer) {
	binary.Write(file, binary.BigEndian, press[:])
}

func main() {
	var (
		press   [8]byte = [8]byte{0x00, 0x00, 0x51, 0x00, 0x00, 0x00, 0x00, 0x00} // down
		press1  [8]byte = [8]byte{0x00, 0x00, 0x2a, 0x00, 0x00, 0x00, 0x00, 0x00} // backspace
		press2  [8]byte = [8]byte{0x00, 0x00, 0x17, 0x00, 0x00, 0x00, 0x00, 0x00} // t
		unpress [8]byte = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	)

	file, err := os.OpenFile("/dev/hidg0", os.O_WRONLY, os.ModePerm)

	fmt.Println(err)
	for j := 1; j <= 1000; j++ {
		Press(press, file)
		Press(unpress, file)
		Press(press1, file)
		Press(unpress, file)
		Press(press2, file)
		Press(unpress, file)
	}

	file.Close()

}
