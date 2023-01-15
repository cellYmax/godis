package main

type Node struct {
	prev *Node
	next *Node
	Val  *Gobj
}

type ListType struct {
	EqualFunc func(a, b *Gobj) bool
}

type List struct {
	head   *Node
	tail   *Node
	length int
	ListType
}

func listCreate(listType ListType) *List {
	var list List
	list.ListType = listType
	return &list
}

func (list *List) AddNodeHead(value *Gobj) {
	var node Node
	node.Val = value
	if list.head == nil {
		list.head = &node
		list.tail = &node
	} else {
		node.next = list.head
		list.head.prev = &node
		list.head = &node
	}
	list.length++
}

func (list *List) AddNodeTail(val *Gobj) {
	var node Node
	node.Val = val
	if list.head == nil {
		list.head = &node
		list.tail = &node
	} else {
		node.prev = list.tail
		list.tail.next = &node
		list.tail = list.tail.next
	}
	list.length++
}

func (list *List) DelNode(node *Node) {
	if node == nil {
		return
	}
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		list.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		list.tail = node.prev
	}
	list.length--
}

func (list *List) Length() int {
	return list.length
}

func (list *List) First() *Node {
	return list.head
}

func (list *List) Last() *Node {
	return list.tail
}

func (list *List) Find(key *Gobj) *Node {
	p := list.head
	for p != nil {
		if list.EqualFunc(key, p.Val) {
			break
		}
		p = p.next
	}
	return p
}

func (list *List) Delete(key *Gobj) {
	list.DelNode(list.Find(key))
}
