package main

import (
	"fmt"
	"testing"
)

func TestGodisCli(t *testing.T) {
	var config cliConfig
	config.ip = [4]byte{127, 0, 0, 1}
	config.port = 4399
	cliRefreshPrompt()
	fmt.Println(string(config.prompt))
}
