package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	list := listCreate(ListType{EqualFunc: GStrEqual})
	assert.Equal(t, list.Length(), 0)

	list.AddNodeTail(CreateObject(GODIS_STRING, "4"))
	list.DelNode(list.First())

	list.AddNodeTail(CreateObject(GODIS_STRING, "1"))
	list.AddNodeTail(CreateObject(GODIS_STRING, "2"))
	list.AddNodeTail(CreateObject(GODIS_STRING, "3"))
	assert.Equal(t, list.Length(), 3)
	assert.Equal(t, list.First().Val.Val.(string), "1")
	assert.Equal(t, list.Last().Val.Val.(string), "3")

	o := CreateObject(GODIS_STRING, "0")
	list.AddNodeHead(o)
	assert.Equal(t, list.Length(), 4)
	assert.Equal(t, list.First().Val.Val.(string), "0")

	list.AddNodeHead(CreateObject(GODIS_STRING, "-1"))
	assert.Equal(t, list.Length(), 5)
	n := list.Find(o)
	assert.Equal(t, n.Val, o)

	list.Delete(o)
	assert.Equal(t, list.Length(), 4)
	n = list.Find(o)
	assert.Nil(t, n)

	list.DelNode(list.First())
	assert.Equal(t, list.Length(), 3)
	assert.Equal(t, list.First().Val.Val.(string), "1")

	list.DelNode(list.Last())
	assert.Equal(t, list.Length(), 2)
	assert.Equal(t, list.Last().Val.Val.(string), "2")
}
