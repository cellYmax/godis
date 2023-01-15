package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func WriteProc(loop *aeEventLoop, fd int, extra interface{}) {
	buf := extra.([]byte)
	n, err := Write(fd, buf)
	if err != nil {
		fmt.Printf("write err: %v\n", err)
		return
	}
	fmt.Printf("ae write %v bytes\n", n)
	loop.RemoveFileEvent(fd, AE_WRITABLE)
}

func ReadProc(loop *aeEventLoop, fd int, extra interface{}) {
	buf := make([]byte, 10)
	n, err := Read(fd, buf)
	if err != nil {
		fmt.Printf("read err: %v\n", err)
		return
	}
	fmt.Printf("ae read %v bytes\n", n)
	loop.AddFileEvent(fd, AE_WRITABLE, WriteProc, buf)
}

func AcceptProc(loop *aeEventLoop, fd int, extra interface{}) {
	cfd, err := Accept(fd)
	if err != nil {
		fmt.Printf("accept err: %v\n", err)
		return
	}
	loop.AddFileEvent(cfd, AE_READABLE, ReadProc, nil)
}
func OnceProc(loop *aeEventLoop, id int, extra interface{}) {
	t := extra.(*testing.T)
	assert.Equal(t, 1, id)
	fmt.Printf("time event %v done\n", id)
}

func NormalProc(loop *aeEventLoop, id int, extra interface{}) {
	end := extra.(chan struct{})
	fmt.Printf("time event %v done\n", id)
	end <- struct{}{}
}
func TestAe(t *testing.T) {
	eventLoop, err := AeCreateEventLoop()
	assert.Nil(t, err)
	sfd, err := TcpServer(6666)
	eventLoop.AddFileEvent(sfd, AE_READABLE, AcceptProc, nil)
	go eventLoop.AeMain()
	host := [4]byte{0, 0, 0, 0}
	cfd, err := Connect(host, 6666)
	assert.Nil(t, err)
	msg := "helloworld"
	n, err := Write(cfd, []byte(msg))
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	buf := make([]byte, 10)
	n, err = Read(cfd, buf)
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, msg, string(buf))
	eventLoop.AddTimeEvent(AE_ONCE, 10, OnceProc, t)
	end := make(chan struct{}, 2)
	eventLoop.AddTimeEvent(AE_NORMAL, 10, NormalProc, end)
	<-end
	<-end
	eventLoop.stop = true
}
