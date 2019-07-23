package main

import (
	"log"
	"os"

	"github.com/hashicorp/tf-sdk-migrator/cmd/check"
	"github.com/mitchellh/cli"
)

func main() {
	// log.SetFlags(log.Llongfile)
	c := cli.NewCLI("tf-sdk-migrator", "0.1.0")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"check": check.CommandFactory,
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}
