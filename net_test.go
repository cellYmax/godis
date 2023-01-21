package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"godis/net"
	"testing"
	"time"
)

func EchoServer(s, c, e chan struct{}) {
	sfd, err := net.TcpServer(6666)
	if err != nil {
		fmt.Printf("tcp server err: %v\n", err)
	}
	fmt.Printf("TcpServer start,serverfd: %v\n", sfd)
	s <- struct{}{}
	<-c
	cfd, err := net.Accept(sfd)
	fmt.Printf("accepted cfd: %v\n", cfd)
	if err != nil {
		fmt.Printf("server accpet error: %v\n", err)
	}
	buf := make([]byte, 10)
	n, err := net.Read(cfd, buf)
	if err != nil {
		fmt.Printf("server read error: %v\n", err)
	}
	fmt.Printf("read %v bytes\n", n)
	n, err = net.Write(cfd, buf)
	if err != nil {
		fmt.Printf("server write error: %v\n", err)
	}
	fmt.Printf("write %v bytes\n", n)
	e <- struct{}{}
}

func TestNet(t *testing.T) {
	s := make(chan struct{})
	c := make(chan struct{})
	e := make(chan struct{})
	go EchoServer(s, c, e)
	<-s
	host := [4]byte{127, 0, 0, 1}
	cfd, err := net.Connect(host, 6666)
	fmt.Printf("connected cfd: %v\n", cfd)
	time.Sleep(100 * time.Millisecond)
	c <- struct{}{}
	assert.Nil(t, err)
	msg := "helloworld"
	n, err := net.Write(cfd, []byte(msg))
	fmt.Printf("write buf: %v\n", []byte(msg))
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	<-e
	buf := make([]byte, 10)
	n, err = net.Read(cfd, buf)
	fmt.Printf("read stop buf: %v\n", buf)
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, msg, string(buf))
}
