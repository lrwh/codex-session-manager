package main

import (
	"log"

	"github.com/liurui/codex-session-manager/internal/cli"
)

var version = "0.2.1"

func main() {
	cli.SetVersion(version)
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
