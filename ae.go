package main

import (
	"log"
	"syscall"
	"time"
)

type FeType int

type TeType int

const (
	AE_READABLE FeType = 1
	AE_WRITABLE FeType = 2
)

const (
	AE_NORMAL TeType = 1
	AE_ONCE   TeType = 2
)

type aeFileProc func(eventLoop *aeEventLoop, fd int, extra interface{})
type aeTimeProc func(eventLoop *aeEventLoop, id int, extra interface{})

type AeFileEvent struct {
	fd       int
	mask     FeType
	fileProc aeFileProc
	extra    interface{}
}

type AeTimeEvent struct {
	id       int
	mask     TeType
	when     int64
	interval int64
	timeProc aeTimeProc
	extra    interface{}
	next     *AeTimeEvent
}

type aeEventLoop struct {
	FileEvents      map[int]*AeFileEvent
	FileEventFd     int
	TimeEvents      *AeTimeEvent
	timeEventNextId int
	stop            bool
}

// var fe2kq = [3]int{0, syscall.EVFILT_READ, syscall.EVFILT_WRITE}

func (eventLoop *aeEventLoop) getKqueueMask(fd int) int {

	if eventLoop.FileEvents[getFeKey(fd, AE_READABLE)] != nil {
		return syscall.EVFILT_READ
	}
	if eventLoop.FileEvents[getFeKey(fd, AE_WRITABLE)] != nil {
		return syscall.EVFILT_WRITE
	}
	return 0
}

func getFeKey(fd int, mask FeType) int {
	if mask == AE_READABLE {
		return fd
	} else {
		return fd * -1
	}
}

func AeCreateEventLoop() (*aeEventLoop, error) {
	kqfd, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}
	return &aeEventLoop{
		FileEvents:      make(map[int]*AeFileEvent),
		FileEventFd:     kqfd,
		timeEventNextId: 1,
		stop:            false}, nil

}

func (eventLoop *aeEventLoop) AddFileEvent(fd int, mask FeType, proc aeFileProc, extra interface{}) {
	//add kevent
	var ke syscall.Kevent_t
	var n int
	var err error
	if mask&AE_WRITABLE != 0 {
		syscall.SetKevent(&ke, fd, syscall.EVFILT_WRITE, syscall.EV_ADD)
		n, err = syscall.Kevent(eventLoop.FileEventFd, []syscall.Kevent_t{ke}, nil, nil)
	}
	if mask&AE_READABLE != 0 {
		syscall.SetKevent(&ke, fd, syscall.EVFILT_READ, syscall.EV_ADD)
		n, err = syscall.Kevent(eventLoop.FileEventFd, []syscall.Kevent_t{ke}, nil, nil)
	}

	if err != nil || n == -1 {
		log.Printf("kqueue add err: %v\n", err)
		return
	}
	//add ae
	var fe AeFileEvent
	fe.fd = fd
	fe.mask = mask
	fe.fileProc = proc
	fe.extra = extra
	eventLoop.FileEvents[getFeKey(fd, mask)] = &fe
	log.Printf("ae add fileEvent fd:%v, mask:%v\n", fd, mask)
}

func (eventLoop *aeEventLoop) RemoveFileEvent(fd int, mask FeType) {
	//remove
	var ke syscall.Kevent_t
	if mask&AE_WRITABLE != 0 {
		syscall.SetKevent(&ke, fd, syscall.EVFILT_WRITE, syscall.EV_DELETE)
	}
	if mask&AE_READABLE != 0 {
		syscall.SetKevent(&ke, fd, syscall.EVFILT_READ, syscall.EV_DELETE)
	}
	n, err := syscall.Kevent(eventLoop.FileEventFd, []syscall.Kevent_t{ke}, nil, nil)

	if err != nil || n == -1 {
		log.Printf("kqueue del err: %v\n", err)
		return
	}
	//remove ae
	eventLoop.FileEvents[getFeKey(fd, mask)] = nil
	log.Printf("ae remove fileEvent fd:%v, mask:%v\n", fd, mask)
}

func (eventLoop *aeEventLoop) AddTimeEvent(mask TeType, interval int64, proc aeTimeProc, extra interface{}) int {
	id := eventLoop.timeEventNextId
	eventLoop.timeEventNextId++
	var te AeTimeEvent
	te.id = id
	te.mask = mask
	te.interval = interval
	te.when = GetMsTime() + interval
	te.timeProc = proc
	te.extra = extra
	te.next = eventLoop.TimeEvents
	eventLoop.TimeEvents = &te
	return id
}

func (eventLoop *aeEventLoop) RemoveTimeEvent(id int) {
	te := eventLoop.TimeEvents
	var prev *AeTimeEvent
	for te != nil {
		if te.id == id {
			if prev == nil {
				eventLoop.TimeEvents = te.next
			} else {
				prev.next = te.next
			}
			te.next = nil
			break
		}
		prev = te
		te = te.next
	}
}

func (eventLoop *aeEventLoop) aeSearchNearestTime() int64 {
	var nearest int64 = GetMsTime() + 1000
	te := eventLoop.TimeEvents
	for te != nil {
		if te.when < nearest {
			nearest = te.when
		}
		te = te.next
	}
	return nearest
}

func (eventLoop *aeEventLoop) AeWait() (fes []*AeFileEvent, tes []*AeTimeEvent) {
	timeout := eventLoop.aeSearchNearestTime() - GetMsTime()

	if timeout <= 0 {
		timeout = 10 //at least wait 10ms
	}
	var events [128]syscall.Kevent_t
	var timespec syscall.Timespec
	timespec.Nsec = timeout * 1000
	n, err := syscall.Kevent(eventLoop.FileEventFd, nil, events[:], &timespec)
	if err != nil {
		//log.Printf("kevent wait err: %v\n", err)
	}
	if n > 0 {
		log.Printf("get %v kevents\n", n)
	}
	//collect fileEvent
	for i := 0; i < n; i++ {
		if events[i].Filter == syscall.EVFILT_READ {
			fe := eventLoop.FileEvents[getFeKey(int(events[i].Ident), AE_READABLE)]
			if fe != nil {
				fes = append(fes, fe)
			}
		}
		if events[i].Filter == syscall.EVFILT_WRITE {
			fe := eventLoop.FileEvents[getFeKey(int(events[i].Ident), AE_WRITABLE)]
			if fe != nil {
				fes = append(fes, fe)
			}
		}
	}

	//collect timeEvent
	now := GetMsTime()
	te := eventLoop.TimeEvents
	for te != nil {
		if te.when <= now {
			tes = append(tes, te)
		}
		te = te.next
	}
	return
}

func GetMsTime() int64 {
	return time.Now().UnixNano() / 1e6
}

func (eventLoop *aeEventLoop) AeProcess(fes []*AeFileEvent, tes []*AeTimeEvent) {
	//te
	for _, te := range tes {
		te.timeProc(eventLoop, te.id, te.extra)
		if te.mask == AE_ONCE {
			eventLoop.RemoveTimeEvent(te.id)
		} else {
			te.when = GetMsTime() + te.interval
		}
	}
	for _, fe := range fes {
		log.Printf("ae is proccessing fileEvents")
		fe.fileProc(eventLoop, fe.fd, fe.extra)
	}
}

func (eventLoop *aeEventLoop) AeMain() {
	for eventLoop.stop != true {
		fes, tes := eventLoop.AeWait()
		if len(fes) > 0 {
			log.Printf("ae wait,get %v file kevents\n", len(fes))
		}
		eventLoop.AeProcess(fes, tes)
	}
}
