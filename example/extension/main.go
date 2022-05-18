/*
This program shows the params structure which implements the Extender interface for validation and modification
of the loaded parameters as well as the fields without an associated flag
*/

package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/matusvla/easyflag"
)

type params struct {
	Username string `flag:"u|Username||required"`
	isAdmin  bool
}

func (p *params) Extend() error {
	// validation of username
	if strings.Index(p.Username, " ") != -1 {
		return errors.New("username cannot contain whitespaces")
	}
	// modification of the structure fields
	if p.Username == "admin" {
		p.isAdmin = true
	}
	return nil
}

func main() {
	// Flag parsing and validation
	var p params
	if err := easyflag.ParseAndLoad(&p); err != nil {
		log.Fatalf("error while parsing the cli parameters: %s", err.Error())
	}

	// The program "logic"
	priviledgesClause := "without admin priviledges"
	if p.isAdmin {
		priviledgesClause = "with admin priviledges"
	}
	fmt.Printf("Running the program as a user %q %s", p.Username, priviledgesClause)
}
