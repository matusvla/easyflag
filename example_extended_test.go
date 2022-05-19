package easyflag

import (
	"errors"
	"log"
	"strings"
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

// Example_extension demonstrates the usage of params structure implementing the Extender interface.
func Example_extension() {
	var p params
	if err := ParseAndLoad(&p); err != nil {
		log.Fatalf("error while parsing the cli parameters: %s", err.Error())
	}
}
