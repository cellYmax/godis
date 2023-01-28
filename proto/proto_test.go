package proto

import (
	"fmt"
	"testing"
)

func TestProto(t *testing.T) {
	line := "set key val"
	msg := "*3\r\n$3\r\nset\r\n$3\r\nkey\r\n$3\r\nval\r\n"
	a := fmt.Sprintf("*%d\r\n", 3)
	fmt.Println(a)
	fmt.Println(len(msg))
	fmt.Println([]byte(msg))
	fmt.Println(len([]byte(msg)))
	cmd := FormatCommandArgs([]byte(line))
	fmt.Println(cmd)
	fmt.Println(len(cmd))
	fmt.Println(CRLF)
	fmt.Println(len(CRLF))
}
