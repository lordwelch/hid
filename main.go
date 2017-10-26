package main

import (
	"fmt"
	"os"
)

func int main() {
	var (
		err error
		input bufio.reader
		key rune;
	)

	input = bufio.NewReader(os.Stdin)

	for err == nil {
		key,_,err = input.ReadRune()
		
}
