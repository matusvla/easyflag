/*
This is a simple program reading n bytes from a file and copying them to the stdout
to illustrate the most basic usage of the easyflag package.

There are two basic flags defined in the params structure: input path (-in/--in) and output length (-n/--n).
The -in flag is required, the -n flag is optional and defaults to the value -1.

-h or -help flags can be used for printing the description of all the flags.
*/

package main

import (
	"io"
	"log"
	"os"

	"github.com/matusvla/easyflag"
)

type params struct {
	InputPath string `flag:"in|Path to the input file||required"`
	OutputLen int64  `flag:"n|Maximum number of characters to read (-1 for all)|-1"`
}

func main() {
	// Flag parsing and validation
	var p params
	if err := easyflag.ParseAndLoad(&p); err != nil {
		log.Fatalf("error while parsing the cli parameters: %s", err.Error())
	}

	// The program "logic"
	f, err := os.Open(p.InputPath)
	if err != nil {
		log.Fatalf("error while opening the input file on path %s: %s", p.InputPath, err.Error())
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalf("error closing the input file: %s", err.Error())
		}
	}()

	if p.OutputLen == -1 {
		if _, err := io.Copy(os.Stdout, f); err != nil {
			log.Fatalf("error writing to stdout: %s", err.Error())
		}
		return
	}

	if _, err := io.CopyN(os.Stdout, f, p.OutputLen); err != nil {
		log.Fatalf("error writing to stdout: %s", err.Error())
	}
}
