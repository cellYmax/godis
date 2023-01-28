package main

import (
	"errors"
	"fmt"
	"godis/net"
	"hash/fnv"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	GODIS_IOBUF_LEN  = 1024 * 16
	GODIS_MAX_BULK   = 1024 * 4
	GODIS_MAX_INLINE = 1024 * 4
)

const (
	GODIS_REQ_UNKNOW    = 0
	GODIS_REQ_INLINE    = 1
	GODIS_REQ_MULTIBULK = 2
)

type GodisCommandProc func(c *GodisClient)

type GodisCommand struct {
	name  string           // 命令名字
	proc  GodisCommandProc //命令实现函数
	arity int              //命令的参数个数
}

type GodisDB struct {
	data   *Dict
	expire *Dict
}

type GodisClient struct {
	fd       int
	db       *GodisDB
	args     []*Gobj
	queryBuf []byte
	queryLen int //queryBuf中未处理的命令的长度
	reqType  int
	bulkNum  int // 读入命令的参数个数
	bulkLen  int // 命令内容的长度
	reply    *List
	sentLen  int // 已发送字节，处理 short write 用
}

type GodisServer struct {
	port    int
	fd      int
	el      *aeEventLoop
	db      *GodisDB
	clients map[int]*GodisClient
}

func GStrEqual(a *Gobj, b *Gobj) bool {
	if a.Type != GODIS_STRING || b.Type != GODIS_STRING {
		return false
	}
	return a.StrVal() == b.StrVal()
}

func GStrHash(key *Gobj) int64 {
	if key.Type != GODIS_STRING {
		return 0
	}
	hash := fnv.New64()
	hash.Write([]byte(key.StrVal()))
	return int64(hash.Sum64())
}

var server GodisServer
var GodisCommandTable []GodisCommand = []GodisCommand{
	{"get", getCommand, 2},
	{"set", setCommand, -3},
	{"expire", expireCommand, 3},
	{"setnx", setnxCommand, 3},
	//TODO
	//{"del",delCommand, -2},
	//{"strlen",strlenCommand, 2},
	//{"incr",incrCommand, 2},
	//{"decr",decrCommand, 2},
	//{"del",delCommand, -2},
	//{"lpush",lpushCommand, -3},
	//{"rpush",rpushCommand, -3},
	//{"sadd",saddCommand, -3},
	//{"zadd",zaddCommand, -4},
	//{"hset",hsetCommand, 4},
	//{"hget",hgetCommand, 3},
	//{"auth",authCommand, 2},
	//{"ping",pingCommand, 1},
	//{"echo",echoCommand, 2},
}

func setnxCommand(c *GodisClient) {
	setGenericCommand(c, REDIS_SET_NX, c.args[1], c.args[2], nil, 0)
}

func expireCommand(c *GodisClient) {
	key := c.args[1]
	val := c.args[2]
	if val.Type != GODIS_STRING {
		//TODO: extract shared.strings
		c.AddReplyStr("-ERR: wrong type\r\n")
	}
	expire := GetMsTime() + (val.IntVal() * 1000)
	expireObj := CreateFromInt(expire)
	server.db.expire.Set(key, expireObj)
	expireObj.DecrRefCount()
	c.AddReplyStr("+OK\r\n")
}

const (
	UNIT_SECONDS = iota
	UNIT_MILLISECONDS
)

const (
	REDIS_SET_NO_FLAGS = 0
	REDIS_SET_NX       = 1 << 0 //Set if key not exists
	REDIS_SET_XX       = 1 << 1 //Set if key exists
)

func setCommand(c *GodisClient) {
	var expire *Gobj
	unit := UNIT_SECONDS
	flags := REDIS_SET_NO_FLAGS
	for i := 3; i < len(c.args); i++ {
		var next *Gobj
		if i == len(c.args)-1 {
			next = nil
		} else {
			next = c.args[i+1]
		}

		str := c.args[i].StrVal()
		a := []byte(str)
		if len(a) == 2 {
			if (a[0] == 'n' || a[0] == 'N') &&
				(a[1] == 'x' || a[1] == 'X') {
				flags |= REDIS_SET_NX
			} else if (a[0] == 'x' || a[0] == 'N') &&
				(a[1] == 'x' || a[1] == 'X') {
				flags |= REDIS_SET_XX
			} else if (a[0] == 'e' || a[0] == 'E') &&
				(a[1] == 'x' || a[1] == 'X') && next != nil {
				unit = UNIT_SECONDS
				expire = next
				i++
			} else if (a[0] == 'p' || a[0] == 'P') &&
				(a[1] == 'x' || a[1] == 'X') && next != nil {
				unit = UNIT_MILLISECONDS
				expire = next
				i++
			}
		} else {
			c.AddReplyStr("-ERR syntax error\r\n")
		}
	}
	//key := c.args[1]
	//val := c.args[2]
	//if val.Type != GODIS_STRING {
	//	c.AddReplyStr("-ERR: wrong type\r\n")
	//}
	//server.db.data.Set(key, val)
	//server.db.expire.Delete(key)
	//c.AddReplyStr("+OK\r\n")
	setGenericCommand(c, flags, c.args[1], c.args[2], expire, unit)
}

func setGenericCommand(c *GodisClient, flags int, key *Gobj, val *Gobj, expire *Gobj, unit int) {
	var milliseconds int64 = 0
	if expire != nil {
		milliseconds = expire.IntVal()
		if milliseconds < 0 {
			c.AddReplyStr("invalid expire time in SETEX")
			return
		}
		if unit == UNIT_SECONDS {
			milliseconds *= 1000
		}
	}
	if (flags&REDIS_SET_NX != 0) && FindKeyWrite(key) != nil ||
		(flags&REDIS_SET_XX != 0) && FindKeyWrite(key) == nil {
		c.AddReplyStr("$-1\r\n")
		return
	}
	server.db.data.Set(key, val)
	server.db.expire.Delete(key)
	if expire != nil {
		expireTime := GetMsTime() + milliseconds
		expireObj := CreateFromInt(expireTime)
		server.db.expire.Set(key, expireObj)
		expireObj.DecrRefCount()
	}
	c.AddReplyStr("+OK\r\n")
}

func getCommand(c *GodisClient) {
	key := c.args[1]
	val := FindKeyRead(key)
	if val == nil {
		c.AddReplyStr("$-1\r\n")
	} else if val.Type != GODIS_STRING {
		c.AddReplyStr("-ERR: wrong type\r\n")
	} else {
		str := val.StrVal()
		c.AddReplyStr(fmt.Sprintf("$%d%v\r\n", len(str), str))
	}

}

func FindKeyRead(key *Gobj) *Gobj {
	//TODO:expireIfNeeded()
	//TODO:更新命中/不命中信息
	return server.db.data.Get(key)
}

func FindKeyWrite(key *Gobj) *Gobj {
	//TODO:expireIfNeeded()
	return server.db.data.Get(key)
}

func (c *GodisClient) findLineInQuery() (int, error) {
	index := strings.Index(string(c.queryBuf[:c.queryLen]), "\r\n")
	if index < -1 && c.queryLen > GODIS_MAX_INLINE {
		return index, errors.New("Protocol error: too big mbulk count string")
	}
	return index, nil
}

// 获取queryBuf中的数字
func (c *GodisClient) getNumInQuery(s, e int) (int, error) {
	num, err := strconv.Atoi(string(c.queryBuf[s:e]))
	c.queryBuf = c.queryBuf[e+2:]
	c.queryLen -= e + 2
	return num, err
}

func freeArgs(c *GodisClient) {
	for _, v := range c.args {
		v.DecrRefCount()
	}
}

func freeReplyList(c *GodisClient) {
	if c.reply.Length() != 0 {
		n := c.reply.head
		c.reply.DelNode(n)
		n.Val.DecrRefCount()
	}
}

func resetClient(c *GodisClient) {
	freeArgs(c)
	c.reqType = GODIS_REQ_UNKNOW
	c.queryBuf = c.queryBuf[0:0]
	c.bulkLen = 0
	c.bulkNum = 0
}

func freeClient(c *GodisClient) {
	/* Close socket, unregister kevents, and remove list of replies and
	 * accumulated arguments. */
	// 关闭套接字，并从事件处理器中删除该套接字的事件
	freeArgs(c)
	freeReplyList(c)
	delete(server.clients, c.fd)
	server.el.RemoveFileEvent(c.fd, AE_READABLE)
	server.el.RemoveFileEvent(c.fd, AE_WRITABLE)
	net.Close(c.fd)
}

func (c *GodisClient) AddReplyStr(str string) {
	o := CreateObject(GODIS_STRING, str)
	c.AddReply(o)
	o.DecrRefCount()
}

func (c *GodisClient) AddReply(o *Gobj) {
	c.reply.AddNodeTail(o)
	o.IncrRefCount()
	server.el.AddFileEvent(c.fd, AE_WRITABLE, sendReplyToClient, c)
}

func sendReplyToClient(el *aeEventLoop, fd int, extra interface{}) {
	c := extra.(*GodisClient)
	log.Printf("SendReplyToClient, reply len:%v\n", c.reply.Length())
	for c.reply.Length() > 0 {
		// 取出位于链表最前面的对象
		rep := c.reply.First()
		buf := []byte(rep.Val.StrVal())
		bufLen := len(buf)
		if c.sentLen < bufLen {
			// 写入内容到套接字
			n, err := net.Write(fd, buf)
			if err != nil {
				log.Printf("send reply err: %v\n", err)
				freeClient(c)
				return
			}
			c.sentLen += n
			log.Printf("send %v bytes to client:%v\n", n, c.fd)
			if c.sentLen == bufLen { // 如果缓冲区内容全部写入完毕，那么删除已写入完毕的节点
				c.reply.DelNode(rep)
				rep.Val.DecrRefCount()
				c.sentLen = 0
			} else {
				break
			}
		}
	}
	if c.reply.Length() == 0 {
		c.sentLen = 0
		el.RemoveFileEvent(fd, AE_WRITABLE)
	}
}

func readQueryFromClient(el *aeEventLoop, fd int, extra interface{}) {
	//client read
	c := extra.(*GodisClient)
	if len(c.queryBuf)-c.queryLen < GODIS_MAX_BULK {
		c.queryBuf = append(c.queryBuf, make([]byte, GODIS_MAX_BULK)...)
	}
	n, err := net.Read(fd, c.queryBuf)
	if err != nil {
		log.Printf("client %v read err: %v\n", fd, err)
		freeClient(c)
		return
	}
	c.queryLen += n
	log.Printf("read %v bytes from client:%v\n", n, c.fd)
	log.Printf("ReadQueryFromClient, queryBuf : %v\n", string(c.queryBuf))
	err = processQueryBuf(c)
	if err != nil {
		log.Printf("process query buf err: %v\n", err)
		freeClient(c)
		return
	}
}

func processMultiBulkBuffer(c *GodisClient) (bool, error) {
	// 第一次读取bulk
	if c.bulkNum == 0 {
		// 获取第一个'\r\n'
		newLine, err := c.findLineInQuery()
		if newLine < 0 {
			return false, err
		}
		// 获取读入命令的参数个数
		bnum, err := c.getNumInQuery(1, newLine)
		if err != nil {
			return false, err
		}
		if bnum == 0 {
			return true, nil
		}
		c.bulkNum = bnum
		c.args = make([]*Gobj, bnum)
	}
	//从c.queryBuf中读入参数并创建对象加到c.args中
	for c.bulkNum > 0 {
		if c.bulkLen == 0 {
			newLine, err := c.findLineInQuery()
			if newLine < 0 {
				return false, err
			}
			if c.queryBuf[0] != '$' {
				return false, errors.New("Protocol error: expected '$' for bulk length")
			}

			blen, err := c.getNumInQuery(1, newLine)
			if err != nil || blen == 0 {
				return false, err
			}
			if blen > GODIS_MAX_BULK {
				return false, errors.New("Protocol error: too big bulk length")
			}
			c.bulkLen = blen
		}
		/* Read bulk argument */
		// 读入参数(string)
		if c.queryLen < c.bulkLen+2 {
			// 没有充足的数据直接返回
			return false, nil
		}
		pos := c.bulkLen
		if c.queryBuf[pos] != '\r' || c.queryBuf[pos+1] != '\n' {
			return false, errors.New("Protocol error: expected CRLF for bulk end")
		}
		c.args[len(c.args)-c.bulkNum] = CreateObject(GODIS_STRING, string(c.queryBuf[:pos]))
		c.queryBuf = c.queryBuf[pos+2:]
		c.queryLen -= pos + 2
		c.bulkLen = 0
		c.bulkNum -= 1
	}

	//完成了所有bulk命令的read
	return true, nil
}

func processInlineBuffer(c *GodisClient) (bool, error) {
	//找到\r\n
	newLine, err := c.findLineInQuery()
	if newLine < 0 {
		return false, err
	}
	//用空格分离Inline命令
	subs := strings.Split(string(c.queryBuf[:newLine]), " ")
	c.queryBuf = c.queryBuf[newLine+2:]
	c.queryLen -= newLine + 2
	c.args = make([]*Gobj, len(subs))
	for i, v := range subs {
		c.args[i] = CreateObject(GODIS_STRING, v)
	}
	return true, nil
}

func processQueryBuf(c *GodisClient) error {
	for len(c.queryBuf) > 0 {
		if c.reqType == GODIS_REQ_UNKNOW {
			if c.queryBuf[0] == '*' {
				// MultiBulk命令
				c.reqType = GODIS_REQ_MULTIBULK
			} else {
				// Inline命令
				c.reqType = GODIS_REQ_INLINE
			}
		}
		var ok bool
		var err error
		if c.reqType == GODIS_REQ_INLINE {
			ok, err = processInlineBuffer(c)
		} else if c.reqType == GODIS_REQ_MULTIBULK {
			ok, err = processMultiBulkBuffer(c)
		} else {
			return errors.New("Unknown request type")
		}
		if err != nil {
			return err
		}
		if ok {
			if len(c.args) == 0 {
				resetClient(c)
			} else {
				processCommand(c)
			}
		} else {
			//queryBuf read不完整
			break
		}
	}
	return nil
}

func processCommand(c *GodisClient) {
	cmdStr := c.args[0].StrVal()
	if cmdStr == "quit" {
		/* Close connection after entire reply has been sent. */
		// 如果指定了写入之后关闭客户端 FLAG ，那么关闭客户端
		freeClient(c)
		return
	}
	cmd := lookupCommand(cmdStr)
	if cmd == nil {
		c.AddReplyStr("-ERR: unknown command\r\n")
		resetClient(c)
		return
	} else if cmd.arity > 0 && cmd.arity != len(c.args) {
		c.AddReplyStr(fmt.Sprintf("-ERR: wrong number of arguments for %s command\r\n", cmd.name))
		resetClient(c)
		return
	}
	cmd.proc(c)
	resetClient(c)
}

func lookupCommand(cmdStr string) *GodisCommand {
	for _, c := range GodisCommandTable {
		if cmdStr == c.name {
			return &c
		}
	}
	return nil
}

// TODO:run background
func serverCron(eventLoop *aeEventLoop, id int, extra interface{}) {

}

func CreateClient(fd int) *GodisClient {
	var client GodisClient
	client.fd = fd
	client.db = server.db
	client.queryBuf = make([]byte, 0, GODIS_IOBUF_LEN)
	client.reply = listCreate(ListType{EqualFunc: GStrEqual})
	return &client
}

func acceptTcpHandler(eventLoop *aeEventLoop, fd int, extra interface{}) {
	cfd, err := net.Accept(fd)
	if err != nil {
		log.Printf("accept client err: %v\n", err)
		return
	}
	client := CreateClient(cfd)
	server.clients[cfd] = client
	server.el.AddFileEvent(cfd, AE_READABLE, readQueryFromClient, client)
	log.Printf("accept client, fd: %v\n", cfd)
}

func initServer(config *Config) error {
	server.port = config.Port
	server.clients = make(map[int]*GodisClient)
	server.db = &GodisDB{
		data:   DictCreate(DictType{HashFunc: GStrHash, EqualFunc: GStrEqual}),
		expire: DictCreate(DictType{HashFunc: GStrHash, EqualFunc: GStrEqual}),
	}
	var err error
	if server.el, err = AeCreateEventLoop(); err != nil {
		return err
	}
	server.fd, err = net.TcpServer(server.port)
	return err
}

func main() {
	path := os.Args[1]
	config, err := LoadConfig(path)
	if err != nil {
		log.Printf("config error: %v\n", err)
	}

	err = initServer(config)
	if err != nil {
		log.Printf("init server error: %v\n", err)
	}
	server.el.AddFileEvent(server.fd, AE_READABLE, acceptTcpHandler, nil)
	server.el.AddTimeEvent(AE_NORMAL, 100, serverCron, nil)
	log.Printf("godis Server started")
	server.el.AeMain()
}
