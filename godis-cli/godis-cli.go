package main

import (
	"bufio"
	"bytes"
	"fmt"
	"godis/net"
	"godis/proto"
	"log"
	"os"
)

const (
	GODIS_BLOCK     = 0x1
	GODIS_CONNECTED = 0x2
)

const GODIS_IOBUF_LEN = 1024 * 16

type godisContext struct {
	fd     int
	flags  int
	oBuf   []byte
	reader *bufio.Reader
}

type cliConfig struct {
	ip     [4]byte
	port   int
	prompt []byte
}

var config cliConfig
var context *godisContext

func (config *cliConfig) clientIp() string {
	return fmt.Sprintf("%d.%d.%d.%d", config.ip[0], config.ip[1], config.ip[2], config.ip[3])
}
func cliRefreshPrompt() {
	ip := config.clientIp()
	config.prompt = []byte(fmt.Sprintf("%s:%d", ip, config.port))
	config.prompt = []byte(fmt.Sprintf("%s> ", config.prompt))
}

func cliConnect() error {
	var err error
	context, err = godisConnect()
	if err != nil {
		return err
	}
	repl()
	err = cliAuth()
	if err != nil {
		return err
	}
	return nil
}

func godisConnect() (*godisContext, error) {
	fd, err := net.Connect(config.ip, config.port)
	c := godisContextInit(fd)
	c.flags |= GODIS_BLOCK
	if err != nil {
		return nil, err
	}
	c.fd = fd
	c.flags |= GODIS_CONNECTED
	return c, nil
}

func godisContextInit(fd int) *godisContext {
	return &godisContext{
		fd:     fd,
		flags:  0,
		oBuf:   make([]byte, 0, GODIS_IOBUF_LEN),
		reader: bufio.NewReader(os.Stdin),
	}
}

// TODO
func cliAuth() error {
	return nil
}

func repl() {
	for {
		line, _ := context.reader.ReadBytes('\n')
		line = bytes.Replace(line, []byte("\n"), []byte(""), -1)
		//line = bytes.TrimSuffix(line, []byte{'\r', '\n'})
		cliSendCommand(line)
	}
}

func cliSendCommand(args []byte) {
	cmd := proto.FormatCommandArgs(args)
	context.oBuf = append(context.oBuf, cmd...)
	cliReadReply()
}

func cliReadReply() {
	buff, err := godisGetReply()
	if err != nil {
		log.Printf("read reply err: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stdout, string(buff))
}

func godisGetReply() ([]byte, error) {
	var err error
	if len(context.oBuf) != 0 {
		_, err = net.Write(context.fd, context.oBuf)
		context.oBuf = context.oBuf[0:0]
	}
	if err != nil {
		log.Printf("write context oBuf err: %v\n", err)
		return nil, err
	}
	rBuf := make([]byte, GODIS_IOBUF_LEN)
	n, err := net.Read(context.fd, rBuf)
	if n == 0 {
		fmt.Println(config.clientIp()+"> ", "nil")
	}
	return rBuf, err
}

func main() {
	config.ip = [4]byte{127, 0, 0, 1}
	config.port = 4399
	err := cliConnect()
	if err != nil {
		os.Exit(1)
	}
	cliRefreshPrompt()
	repl()
}
