package linkedbuf

import (
	"container/list"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

const BLOCKSIZE int = 2

const (
	RefCountAdd   = 0
	RefCountMinus = 1
)

var blockPool sync.Pool

type Block struct {
	refCount   int32
	data       []byte
	blockIndex int
}

func (b *Block) String() string {
	return fmt.Sprintf("blockInex:%d,refCount:%d", b.blockIndex, b.refCount)
}

func (b *Block) reset(blockIndex int) {
	b.refCount = 0
	b.blockIndex = blockIndex
}

type Segment struct {
	block  *Block
	b      []byte
	droped bool
}

func (d *Segment) Drop() {
	if d.droped {
		return
	}
	refCount(d.block, RefCountMinus)
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
}

func New() *LinkedBuffer {
	l := list.New()
	block := NewBlock(0)
	l.PushBack(block)
	return &LinkedBuffer{
		l: l,
		wp: Point{
			b:   block,
			pos: 0,
		},
		nextBlockIndex: 1,
	}
}

func (buf *LinkedBuffer) growth() {
	//buf.Gc()

	//buf.lock.Lock()
	block := NewBlock(buf.nextBlockIndex)
	buf.l.PushBack(block)
	buf.nextBlockIndex++
	buf.wp.pos = 0
	buf.wp.b = block
	//buf.lock.Unlock()
}

func (buf *LinkedBuffer) NexWriteBlock() []byte {
	if buf.wp.pos == BLOCKSIZE {
		buf.growth()
	}
	return buf.wp.b.data[buf.wp.pos:]
}

func (buf *LinkedBuffer) MoveWritePiont(n int) (s *Segment) {
	s = new(Segment)
	s.block = buf.wp.b
	s.b = buf.wp.b.data[buf.wp.pos : buf.wp.pos+n]
	buf.wp.pos += n
	buf.wp.b.refCount++
	return s
}

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
		if block == buf.wp.b {
			break
		}
		if atomic.LoadInt32(&block.refCount) == 0 {
			buf.l.Remove(item)
			blockPool.Put(block)
		} else {
			break
		}
	}
}

func refCount(b *Block, op int) {
	if op == RefCountAdd {
		atomic.AddInt32(&b.refCount, 1)
	} else {
		atomic.AddInt32(&b.refCount, -1)
	}
}

type ConpositeBuf struct {
	segments []*Segment
	length   int
	read     int
}

func (c *ConpositeBuf) Wrap(s *Segment) {
	c.segments = append(c.segments, s)
	c.length += len(s.b)
}

func (c *ConpositeBuf) Bufferd() int {
	return c.length - c.read
}

func (c *ConpositeBuf) Read(b []byte) (n int, err error) {
	if c.length == 0 {
		return 0, errors.New("no data")
	}
	if c.read >= c.length {
		return 0, io.EOF
	}
	count := len(b)
	if count == 0 {
		return 0, nil
	}

	//found read postion
	var j = 0
	var pos = 0
	var sIndex = 0
	for i, s := range c.segments {
		if j+len(s.b) > c.read {
			sIndex = i
			pos = c.read - j
			break
		}
	}

	//copy data
	for ; sIndex < len(c.segments); sIndex++ {
		block := c.segments[sIndex].b[pos:]
		if n >= count-1 {
			break
		}
		num := copy(b[n:], block)
		c.read += num
		n += num
		pos = 0
	}
	return
}

func (c *ConpositeBuf) ReadN(n int) ([]byte, error) {
	if n <= 0 {
		return nil, errors.New("Parameter error")
	}
	if c.length < n {
		return nil, errors.New("not enough data")
	}

	if n <= len(c.segments[0].b) {
		c.read += n
		return c.segments[0].b[:n], nil
	}
	bigB := make([]byte, n)
	c.Read(bigB)

	return bigB, nil
}

/* func (c *ConpositeBuf) Rest() {
	c.read = 0
} */

func (c *ConpositeBuf) Discurd() {
	for _, s := range c.segments {
		s.Drop()
	}
}
