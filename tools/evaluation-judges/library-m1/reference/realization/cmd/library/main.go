// Command library serves one small library over JSON Lines on standard
// input and standard output. See the README for the wire format.
package main

import (
	"fmt"
	"os"

	"example.com/smalllibrary/internal/protocol"
)

func main() {
	if err := protocol.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "library:", err)
		os.Exit(1)
	}
}
