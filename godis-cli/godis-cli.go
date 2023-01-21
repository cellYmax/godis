package main

import (
	"bufio"
	"fmt"
	"godis/net"
	"os"
)

const (
	REDIS_BLOCK     = 0x1
	REDIS_CONNECTED = 0x2
)

type godisContext struct {
	fd          int
	flags       int
	writeBuf    []byte
	godisReader *bufio.Reader
}

type cliConfig struct {
	ip     [4]byte
	port   int
	prompt []byte
}

var config cliConfig
var context *godisContext

func cliRefreshPrompt() {
	ip := fmt.Sprintf("%d.%d.%d.%d", config.ip[0], config.ip[1], config.ip[2], config.ip[3])
	config.prompt = []byte(fmt.Sprintf("%s:%d", ip, config.port))
	config.prompt = []byte(fmt.Sprintf("%s> ", config.prompt))
}

	var err error
	context, err = godisConnect()
	if err != nil {
		return err
	}
	err = cliAuth()
	if err != nil {
		return err
	}
	return nil
}

func godisConnect() (*godisContext, error) {
	c := godisContextInit()
	c.flags |= REDIS_BLOCK
	//c.godisContextConnectTcp(config.ip, config.port)
	fd, err := net.Connect(config.ip, config.port)
	if err != nil {
		return nil, err
	}
	c.fd = fd
	c.flags |= REDIS_CONNECTED
	return c, nil
}

//func (c *godisContext) godisContextConnectTcp(ip [4]byte, port int) {
//	fd, err := Connect(ip, port)
//	if err != nil {
//		log.Printf("connect error: %v\n", err)
//		return
//	}
//	c.fd = fd
//	c.flags |= REDIS_CONNECTED
//}

func godisContextInit() *godisContext {
	return &godisContext{
		fd:          0,
		flags:       0,
		writeBuf:    nil,
		godisReader: bufio.NewReader(os.Stdin),
	}
}

// TODO
func cliAuth() error {
	return nil
}

func repl() {

}
func main() {
	config.ip = [4]byte{127, 0, 0, 1}
	config.port = 4399
	err := cliConnect()
	if err != nil {
		os.Exit(1)
	}
	repl()
}
