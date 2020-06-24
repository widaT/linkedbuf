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

	_, le := l.Bytes()
	t.Errorf("%d", n)
	l.Shift(le)
	t.Errorf("%d", l.Buffered())
}
