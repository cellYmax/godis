package proto

import (
	"bytes"
	"fmt"
	"strconv"
)

var CRLF = "\r\n"

const (
	GODIS_REPLY_STATUS  = '+'
	GODIS_REPLY_ERROR   = '-'
	GODIS_REPLY_INTEGER = ':'
	GODIS_REPLY_STRING  = '$'
	GODIS_REPLY_ARRAY   = '*'
)

type godisReply struct {
	Type    byte
	Value   []byte
	Element []*godisReply
}

// set key val -> *3\r\n$3\r\nset\r\n ...
func FormatCommandArgs(line []byte) []byte {
	args := bytes.Split(line, []byte{' '})
	argLen := len(args)
	totlen := 1 + intLen(argLen) + 2
	for _, arg := range args {
		totlen += bulkLen(len(arg))
	}
	cmd := make([]byte, totlen)
	cmd = []byte(fmt.Sprintf("*" + strconv.Itoa(argLen) + CRLF))
	for _, arg := range args {
		arg = []byte(fmt.Sprintf("$" + strconv.Itoa(len(arg)) + CRLF + string(arg) + CRLF))
		cmd = []byte(fmt.Sprintf("%s%s", cmd, arg))
	}
	return cmd
}

func intLen(i int) int {
	intlen := 0
	for {
		intlen++
		i /= 10
		if i == 0 {
			break
		}
	}
	return intlen
}

func bulkLen(i int) int {
	return 1 + intLen(i) + 2 + i + 2 //$3/r/nSET/r/n
}
