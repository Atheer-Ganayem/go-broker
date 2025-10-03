package broker

import (
	"fmt"
	"sync"

	"github.com/Atheer-Ganayem/go-broker/internal"
)

type channel struct {
	name     string
	id       uint16
	conns    map[*conn]bool
	mu       sync.RWMutex
	messages chan *internal.Message
}

func newChannel(name string, id uint16) *channel {
	ch := &channel{
		name:     name,
		id:       id,
		conns:    make(map[*conn]bool),
		messages: make(chan *internal.Message, 1024),
	}

	go ch.listener()
	return ch
}

func (ch *channel) addConn(c *conn) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.conns[c] = true
}

func (ch *channel) removeConn(c *conn) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	delete(ch.conns, c)
}

func (ch *channel) Pub(msg *internal.Message) {
	select {
	case ch.messages <- msg:
	default:
		fmt.Println("channel is full")
	}
}

func (ch *channel) listener() {
	for msg := range ch.messages {
		ch.broadcast(msg)
	}
}

func (ch *channel) broadcast(msg *internal.Message) {
	for c := range ch.conns {
		select {
		case c.messages <- msg:
		default:
			fmt.Println("conn full")
		}
	}
}
