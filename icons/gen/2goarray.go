// Simple utility to convert a file into a Go byte array

// Clint Caywood

// http://github.com/cratonica/2goarray
package main

import (
	"fmt"
	"io"
	"os"
)

const (
	name         = "2goarray"
	version      = "0.1.0"
)

func main() {
	if len(os.Args) !=2 {
		fmt.Print(name + " v" + version + "\n\n")
		fmt.Println("Usage: " + name + " array_name")
		return
	}

	fmt.Printf("var %s = []byte {", os.Args[1])
	buf := make([]byte, 1)
	var err error
	var totalBytes uint64
	var n int
	for n, err = os.Stdin.Read(buf); n > 0 && err == nil; {
		if totalBytes%12 == 0 {
			fmt.Printf("\n\t")
		}
		fmt.Printf("0x%02x, ", buf[0])
		totalBytes++
		n, err = os.Stdin.Read(buf)
	}
	if err != nil && err != io.EOF {
		err = fmt.Errorf("Error: %v", err)
	}
	fmt.Print("\n}\n\n")
}
