package main

import "strconv"

type Gval interface {
}

type Gtype uint8

const (
	GODIS_STRING Gtype = 0
	GODIS_LIST   Gtype = 1
	GODIS_SET    Gtype = 2
	GODIS_ZSET   Gtype = 3
	GODIS_DICT   Gtype = 4
)

type Gobj struct {
	Type     Gtype
	Val      Gval
	refCount int
}

func (o *Gobj) StrVal() string {
	if o.Type != GODIS_STRING {
		return ""
	}
	return o.Val.(string)
}

func (o *Gobj) IntVal() int64 {
	if o.Type != GODIS_STRING {
		return 0
	}
	val, _ := strconv.ParseInt(o.Val.(string), 10, 64)
	return val
}

func (o *Gobj) DecrRefCount() {
	o.refCount--
	if o.refCount == 0 {
		o.Val = nil
	}

}
func (o *Gobj) IncrRefCount() {
	o.refCount++
}

func CreateObject(Type Gtype, ptr interface{}) *Gobj {
	return &Gobj{
		Type:     Type,
		Val:      ptr,
		refCount: 1,
	}
}

func CreateFromInt(val int64) *Gobj {
	return &Gobj{
		Type:     GODIS_STRING,
		Val:      strconv.FormatInt(val, 10),
		refCount: 1,
	}
}
