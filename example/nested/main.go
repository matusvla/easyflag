/*
This example shows how you can group the CLI parameters using the substructures
*/

package main

import (
	"fmt"
	"log"

	"github.com/matusvla/easyflag"
)

type userAuth struct {
	Username string `flag:"user|Username||required"`
	Password string `flag:"pass|Passsword"`
}

type serverInfo struct {
	Host string `flag:"a|Server host address|127.0.0.1"`
	Port int    `flag:"p|Server port|80"`
}

type params struct {
	UserAuth   userAuth
	ServerInfo serverInfo
}

func main() {
	// Flag parsing and validation
	var p params
	if err := easyflag.ParseAndLoad(&p); err != nil {
		log.Fatalf("error while parsing the cli parameters: %s", err.Error())
	}

	// The program "logic"
	fmt.Printf("Connecting server at %s:%d, username %s, password: *** (not gonna print that!)", p.ServerInfo.Host, p.ServerInfo.Port, p.UserAuth.Username)
}
