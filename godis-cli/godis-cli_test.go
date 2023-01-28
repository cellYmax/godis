package main

import (
	"testing"
)

func TestGodisCli(t *testing.T) {
	var config cliConfig
	config.ip = [4]byte{127, 0, 0, 1}
	config.port = 4399
	//cliRefreshPrompt()

	//str := "*3\r\n$3\r\nSET\r\n$3\r\nKEY\r\n$3\r\nMSG\r\n"

}
