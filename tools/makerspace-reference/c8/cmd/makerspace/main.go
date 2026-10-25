// Command makerspace speaks JSON Lines on standard input and output.
package main

import (
	"bufio"
	"io"
	"os"

	"makerspace/internal/ms"
)

func main() {
	in := bufio.NewReaderSize(os.Stdin, 1<<16)
	out := bufio.NewWriter(os.Stdout)
	server := ms.NewServer()
	for {
		line, err := in.ReadBytes('\n')
		if len(line) > 0 {
			out.Write(server.Handle(line))
			out.WriteByte('\n')
			out.Flush()
		}
		if err != nil {
			if err != io.EOF {
				os.Stderr.WriteString("read error: " + err.Error() + "\n")
			}
			return
		}
	}
}
