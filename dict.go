package main

import (
	"errors"
	"math"
	"math/rand"
)

const (
	INIT_SIZE           int64 = 4
	GROW_RATIO          int64 = 2
	FORCE_RESIZE_RATION int64 = 2
	DEFAULT_STEP        int   = 1
)

var (
	EXPAND_ERR        = errors.New("expand error")
	KEY_EXIST_ERR     = errors.New("key exists error")
	KEY_NOT_EXIST_ERR = errors.New("key doesn't exist error")
	TABLE_EMPTY_ERR   = errors.New("table empty error")
)

type Dict struct {
	DictType
	ht        [2]*htable
	rehashidx int64
	//iterators
}

type Entry struct {
	key  *Gobj
	val  *Gobj
	next *Entry
}

type DictType struct {
	HashFunc  func(key *Gobj) int64
	EqualFunc func(k1, k2 *Gobj) bool
}

type htable struct {
	table []*Entry // 哈希表数组
	size  int64    // 哈希表大小
	mask  int64    // 哈希表大小掩码，用于计算索引值
	used  int64    // 该哈希表已有节点的数量
}

func DictCreate(dictType DictType) *Dict {
	var dict Dict
	dict.DictType = dictType
	dict.rehashidx = -1
	return &dict
}

func (dict *Dict) isRehashing() bool {
	return dict.rehashidx != -1
}
func (dict *Dict) rehashStep() {
	//TODO: check iterators
	dict.rehash(DEFAULT_STEP)
}

func (dict *Dict) rehash(step int) {
	if !dict.isRehashing() {
		return
	}
	for step > 0 {
		if dict.ht[0].used == 0 {
			dict.ht[0] = dict.ht[1]
			dict.ht[1] = nil
			dict.rehashidx = -1
			return
		}
		// 略过数组中为空的索引，找到下一个非空索引 find an nonull slot
		for dict.ht[0].table[dict.rehashidx] == nil {
			dict.rehashidx++
		}
		entry := dict.ht[0].table[dict.rehashidx]
		for entry != nil {
			ne := entry.next
			idx := dict.HashFunc(entry.key) & dict.ht[1].mask
			entry.next = dict.ht[1].table[idx]
			dict.ht[1].table[idx] = entry
			dict.ht[0].used--
			dict.ht[1].used++
			entry = ne
		}
		dict.ht[0].table[dict.rehashidx] = nil
		dict.rehashidx++
		step--
	}

}

func nextPower(size int64) int64 {
	for i := INIT_SIZE; i < math.MaxInt64; i *= 2 {
		if i >= size {
			return i
		}
	}
	return -1

}

func expand(dict *Dict, size int64) error {
	sz := nextPower(size)
	if dict.isRehashing() || (dict.ht[0] != nil && dict.ht[0].size >= sz) {
		return EXPAND_ERR
	}
	var ht htable
	ht.size = sz
	ht.mask = sz - 1
	ht.table = make([]*Entry, sz)
	ht.used = 0
	if dict.ht[0] == nil { // 初始化
		dict.ht[0] = &ht
		return nil
	} else {
		dict.ht[1] = &ht // rehash
		dict.rehashidx = 0
		return nil
	}
}

func (dict *Dict) expandIfNeeded() error {
	if dict.isRehashing() {
		return nil
	}
	if dict.ht[0] == nil {
		return expand(dict, INIT_SIZE)
	}
	if dict.ht[0].used >= dict.ht[0].size && dict.ht[0].used/dict.ht[0].size > FORCE_RESIZE_RATION {
		return expand(dict, dict.ht[0].size*GROW_RATIO)
	}
	return nil
}

func (dict *Dict) keyIndex(key *Gobj) int64 {
	err := dict.expandIfNeeded()
	if err != nil {
		return -1
	}
	h := dict.HashFunc(key)
	var idx int64
	for table := 0; table <= 1; table++ {
		idx = h & dict.ht[table].mask
		e := dict.ht[table].table[idx]
		for e != nil {
			if dict.EqualFunc(e.key, key) {
				return -1
			}
			e = e.next
		}
		if !dict.isRehashing() {
			break
		}
	}
	return idx
}

func (dict *Dict) AddRaw(key *Gobj) *Entry {
	if dict.isRehashing() {
		dict.rehashStep()
	}
	idx := dict.keyIndex(key)
	if idx == -1 {
		return nil
	}
	var ht *htable
	if dict.isRehashing() {
		ht = dict.ht[1]
	} else {
		ht = dict.ht[0]
	}
	var entry Entry
	entry.key = key
	key.IncrRefCount()
	entry.next = ht.table[idx]
	ht.table[idx] = &entry
	ht.used++
	return &entry
}

// Add a new key-val 添加到dict中，当键值存在时return err
func (dict *Dict) Add(key, val *Gobj) error {
	entry := dict.AddRaw(key)
	if entry == nil {
		return KEY_EXIST_ERR
	}
	entry.val = val
	val.IncrRefCount()
	return nil
}

func (dict *Dict) Set(key, val *Gobj) {
	if err := dict.Add(key, val); err == nil {
		return
	}
	entry := dict.Find(key)
	entry.val.DecrRefCount()
	entry.val = val
	val.IncrRefCount()
}

func (dict *Dict) Find(key *Gobj) *Entry {
	if dict.ht[0] == nil {
		return nil
	}
	if dict.isRehashing() {
		dict.rehashStep()
	}
	h := dict.HashFunc(key)
	for table := 0; table <= 1; table++ {
		idx := h & dict.ht[table].mask
		e := dict.ht[table].table[idx]
		for e != nil {
			if dict.EqualFunc(e.key, key) {
				return e
			}
			e = e.next
		}
		if !dict.isRehashing() {
			break
		}
	}
	return nil
}

func (dict *Dict) Delete(key *Gobj) error {
	if dict.ht[0] == nil {
		return TABLE_EMPTY_ERR
	}
	if dict.isRehashing() {
		dict.rehashStep()
	}
	h := dict.HashFunc(key)
	for table := 0; table <= 1; table++ {
		idx := h & dict.ht[0].mask
		e := dict.ht[0].table[idx]
		var prev *Entry
		for e != nil {
			if dict.EqualFunc(e.key, key) {
				if prev == nil {
					dict.ht[0].table[idx] = e.next
				} else {
					prev.next = e.next
				}
				freeEntry(e)
				dict.ht[0].used--
				return nil
			}
			prev = e
			e = e.next
		}
		if !dict.isRehashing() {
			break
		}
	}
	return KEY_NOT_EXIST_ERR
}

func freeEntry(e *Entry) {
	e.key.DecrRefCount()
	e.val.DecrRefCount()
}

func (dict *Dict) Get(key *Gobj) *Gobj {
	entry := dict.Find(key)
	if entry != nil {
		return entry.val
	}
	return nil
}

func (dict *Dict) RandomGet() *Entry {
	if dict.ht[0] == nil {
		return nil
	}
	if dict.isRehashing() {
		dict.rehashStep()
	}
	var h int64
	var e *Entry
	if dict.isRehashing() {
		for {
			h = dict.rehashidx + rand.Int63n(dict.ht[0].size+dict.ht[1].size-(dict.rehashidx))
			if h >= dict.ht[0].size {
				e = dict.ht[1].table[h-dict.ht[0].size]
			} else {
				e = dict.ht[0].table[h]
			}
			if e != nil {
				break
			}
		}
	} else {
		for {
			h = rand.Int63n(dict.ht[0].size)
			e = dict.ht[0].table[h]
			if e != nil {
				break
			}
		}
	}
	if e == nil {
		return nil
	}
	var listLen int64
	p := e
	for p != nil {
		listLen++
		p = p.next
	}
	listIdx := rand.Int63n(listLen)
	p = e
	for i := int64(0); i < listIdx; i++ {
		p = p.next
	}
	return p
}
