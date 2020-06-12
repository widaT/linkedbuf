package linkedbuf

import (
	"bytes"
	"io"
	"testing"
)

func TestLinkedBuffer(t *testing.T) {

	b := []byte("123456789")

	linkedbuf := New()

	readNum := 0

	c := ConpositeBuf{}

	for {
		if readNum >= len(b) {
			break
		}
		buf := linkedbuf.NexWriteBlock()
		n := copy(buf, b[readNum:])
		s := linkedbuf.MoveWritePiont(n)
		c.Wrap(s)
		readNum += n
	}

	t.Errorf("%d", c.length)

	data := make([]byte, 10)
	n, err := c.Read(data)

	t.Errorf("%s ,%d,%s", data[:n], n, err)

	c.Drop()

	linkedbuf.Gc()

	linkedbuf.Range(func(b *Block) {
		t.Error(b)
	})
}

func TestIoCopy(t *testing.T) {
	b := []byte("123456789")

	linkedbuf := New()

	readNum := 0

	c := &ConpositeBuf{}
	defer c.Drop()
	for {
		if readNum >= len(b) {
			break
		}
		buf := linkedbuf.NexWriteBlock()
		n := copy(buf, b[readNum:])
		s := linkedbuf.MoveWritePiont(n)
		c.Wrap(s)
		readNum += n
	}

	cc := make([]byte, 10)
	var bbuf = bytes.NewBuffer(cc)

	n, err := io.Copy(bbuf, c)

	t.Errorf("%d %s %s", n, bbuf.Bytes(), err)
}
