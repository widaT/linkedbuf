package buf

import (
	"testing"
)

func TestBuf(t *testing.T) {

	l := New()
	c := make([]byte, 98982)
	n := 0
	for {
		if n >= len(c) {
			break
		}
		b := l.NexWritablePos()
		num := copy(b, c[n:])
		l.MoveWritePiont(num)
		n += num
	}

	t.Errorf("%d", l.Buffered())

	le := len(l.Bytes())
	t.Errorf("%d", n)
	l.Shift(le)
	t.Errorf("%d", l.Buffered())
}

func TestWrite(t *testing.T) {

	b := []byte("abcdefghizklmn")

	l := New()

	l.Write(b)

	t.Errorf("%s,%v,%v", l.Bytes(), l.wp.b, l.wp.pos)

}
