package object

import (
	"sync"
)

// Channel is a GoScript channel for goroutine communication
type Channel struct {
	ch     chan Object
	closed bool
	mu     sync.Mutex
}

func (c *Channel) Type() ObjectType { return CHANNEL_OBJ }
func (c *Channel) Inspect() string  { return "<channel>" }

func NewChannel(capacity int) *Channel {
	return &Channel{
		ch:     make(chan Object, capacity),
		closed: false,
	}
}

func (c *Channel) Send(obj Object) bool {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return false
	}
	c.mu.Unlock()
	c.ch <- obj
	return true
}

func (c *Channel) Recv() (Object, bool) {
	obj, ok := <-c.ch
	if !ok {
		return UNDEFINED, false
	}
	return obj, true
}

func (c *Channel) TryRecv() (Object, bool) {
	select {
	case obj, ok := <-c.ch:
		if !ok {
			return UNDEFINED, false
		}
		return obj, true
	default:
		return UNDEFINED, false
	}
}

func (c *Channel) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.ch)
	}
}

func (c *Channel) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}
