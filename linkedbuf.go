package buf

import (
	"container/list"
	"fmt"
	"io"
	"sync"
)

const GcFrequency int = 6
const BLOCKSIZE int = 4096

/* const (
	RefCountAdd   = 0
	RefCountMinus = 1
) */

var blockPool sync.Pool

type Block struct {
	//	refCount   int32
	data       []byte
	blockIndex int
	next       *Block
}

func (b *Block) String() string {
	return fmt.Sprintf("blockInex:%d", b.blockIndex)
}

func (b *Block) reset(blockIndex int) {
	//	b.refCount = 0
	b.blockIndex = blockIndex
	b.next = nil
}

func NewBlock(blockIndex int) *Block {
	b := blockPool.Get()
	if b != nil {
		block := b.(*Block)
		block.reset(blockIndex)
		return block
	}

	return &Block{
		data:       make([]byte, BLOCKSIZE),
		blockIndex: blockIndex,
	}
}

type Point struct {
	b   *Block
	pos int
}

type LinkedBuffer struct {
	l              *list.List
	nextBlockIndex int
	wp             Point
	rp             Point
}

func New() *LinkedBuffer {
	l := list.New()
	block := NewBlock(0)
	l.PushBack(block)
	return &LinkedBuffer{
		l: l,
		rp: Point{
			b:   block,
			pos: 0,
		},
		wp: Point{
			b:   block,
			pos: 0,
		},
		nextBlockIndex: 1,
	}
}

func (buf *LinkedBuffer) growth() {
	if buf.nextBlockIndex%GcFrequency == 0 {
		buf.Gc()
	}
	block := NewBlock(buf.nextBlockIndex)
	buf.wp.b.next = block
	buf.l.PushBack(block)
	buf.nextBlockIndex++
	buf.wp.pos = 0
	buf.wp.b = block
}

func (buf *LinkedBuffer) NexWritablePos() []byte {
	if buf.wp.pos == BLOCKSIZE {
		buf.growth()
	}
	return buf.wp.b.data[buf.wp.pos:]
}

func (buf *LinkedBuffer) MoveWritePiont(n int) {
	buf.wp.pos += n
	//buf.wp.b.refCount++
}

func (buf *LinkedBuffer) Write(b []byte) {
	if buf == nil {
		buf = New()
	}
	wp := buf.wp
	left := BLOCKSIZE - wp.pos
	length := len(b)
	if length <= left {
		buf.wp.pos += copy(wp.b.data[wp.pos:], b)
		return
	}
	copyed := copy(buf.wp.b.data[wp.pos:], b)
	for length-copyed > 0 {
		buf.growth()
		//min(length-copyed, BLOCKSIZE)
		buf.wp.pos = copy(buf.wp.b.data, b[copyed:])
		copyed += buf.wp.pos
	}
}

//Bytes 返回缓存的所有字节 不会移动read位置 在一个block里头，不会拷贝
func (buf *LinkedBuffer) Bytes() ([]byte, int) {
	if buf == nil {
		return nil, 0
	}
	n := buf.Buffered()
	wp := buf.wp
	rp := buf.rp
	left := BLOCKSIZE - rp.pos
	if n <= left {
		return rp.b.data[rp.pos:wp.pos], n
	}
	b := make([]byte, n)
	nn := 0
	nn += copy(b, rp.b.data[rp.pos:])
	block := rp.b
	for block.next != nil {
		block = block.next
		nn += copy(b[nn:], block.data[:min(n-nn, BLOCKSIZE)])
	}
	return b, n
}

//Shift 移动read位置
func (buf *LinkedBuffer) Shift(n int) {
	if n == 0 {
		return
	}
	rp := buf.rp
	left := BLOCKSIZE - rp.pos
	if n <= left {
		buf.rp.pos += n
		return
	}
	if n > buf.Buffered() {
		n = buf.Buffered()
	}
	nn := left
	block := rp.b
	pos := 0
	for block.next != nil {
		if nn >= n {
			break
		}
		block = block.next
		buf.rp.b = block
		pos = min(n-nn, BLOCKSIZE)
		nn += pos
		buf.rp.pos = pos
	}
}

//ReadN 返回缓存中的前n个字节 不会移动read位置，在一个block里头，不会拷贝
func (buf *LinkedBuffer) ReadN(n int) ([]byte, int) {
	if n == 0 {
		return nil, 0
	}
	wp := buf.wp
	rp := buf.rp
	if wp.b == rp.b && wp.pos == rp.pos {
		return nil, 0
	}
	if n > buf.Buffered() {
		n = buf.Buffered()
	}
	nn := 0
	if len(rp.b.data[rp.pos:]) >= n {
		return rp.b.data[rp.pos : rp.pos+n], n
	}

	b := make([]byte, n)
	nn += copy(b, rp.b.data[rp.pos:])
	block := rp.b
	for block.next != nil {
		block = block.next
		nn += copy(b[nn:], block.data[:min(n-nn, BLOCKSIZE)])
	}
	return b, n
}

//Read 返回缓存中的字节拷贝到b 会移动read位置，会发生拷贝
func (buf *LinkedBuffer) Read(b []byte) (n int, err error) {
	n = len(b)
	if n == 0 {
		return
	}
	wp := buf.wp
	rp := buf.rp
	if wp.b == rp.b && wp.pos == rp.pos {
		err = io.EOF
		return
	}
	if n > buf.Buffered() {
		n = buf.Buffered()
	}
	nn := 0
	if len(rp.b.data[rp.pos:]) >= n {
		buf.rp.pos += copy(b, rp.b.data[rp.pos:rp.pos+n])
		return
	}

	nn += copy(b, rp.b.data[rp.pos:])
	block := rp.b
	for block.next != nil {
		block = block.next
		buf.rp.b = block
		buf.rp.pos = copy(b[nn:], block.data[:min(n-nn, BLOCKSIZE)])
		nn += buf.rp.pos
	}
	return
}

//Release 释放所有block，block会被pool回收
func (buf *LinkedBuffer) Release() {
	var next *list.Element
	for item := buf.l.Front(); item != nil; item = next {
		next = item.Next()
		block := item.Value.(*Block)
		buf.l.Remove(item)
		blockPool.Put(block)
	}
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

//Buffered 返回缓存大小
func (buf *LinkedBuffer) Buffered() int {
	if buf == nil {
		return 0
	}
	wp := buf.wp
	rp := buf.rp
	n := wp.b.blockIndex - rp.b.blockIndex
	if n == 0 {
		return wp.pos - rp.pos
	}
	return (BLOCKSIZE - rp.pos + wp.pos) + (n-1)*BLOCKSIZE
}

//BlockLen 返回block个数
func (buf *LinkedBuffer) BlockLen() int {
	return buf.l.Len()
}

func (buf *LinkedBuffer) Range(fn func(*Block)) {
	for item := buf.l.Front(); item != nil; item = item.Next() {
		block := item.Value.(*Block)
		fn(block)
	}
}

func (buf *LinkedBuffer) Gc() {
	var next *list.Element
	for item := buf.l.Front(); item != nil; item = next {
		next = item.Next()
		block := item.Value.(*Block)
		if block == buf.rp.b {
			break
		}
		buf.l.Remove(item)
		blockPool.Put(block)
	}
}

/* func refCount(b *Block, op int) {
	if op == RefCountAdd {
		atomic.AddInt32(&b.refCount, 1)
	} else {
		atomic.AddInt32(&b.refCount, -1)
	}
}
*/
