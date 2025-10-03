package broker

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Atheer-Ganayem/go-broker/internal"
)

type conn struct {
	netConn   net.Conn
	reader    *internal.ConnReader
	closed    chan struct{}
	closeOnce sync.Once

	broker   *broker
	channels map[*channel]bool
	mu       sync.RWMutex

	messages chan *internal.Message
}

func newConn(netConn net.Conn, broker *broker) *conn {
	return &conn{
		netConn:  netConn,
		closed:   make(chan struct{}),
		reader:   internal.NewConnReader(netConn),
		messages: make(chan *internal.Message, 1024),
		broker:   broker,
		channels: make(map[*channel]bool),
	}
}

func (c *conn) listener() {
	defer c.close()
	for msg := range c.messages {
		if err := c.netConn.SetWriteDeadline(time.Now().Add(time.Second * 5)); err != nil {
			fmt.Println("write deadline failed:", err)
			return
		}

		_, err := c.netConn.Write(internal.EncodeMessage(msg))
		if err != nil {
			fmt.Println("write failed:", err)
			return
		}
	}
}

func (c *conn) close() {
	c.closeOnce.Do(func() {
		close(c.closed)

		c.mu.Lock()
		defer c.mu.Unlock()
		for ch := range c.channels {
			ch.removeConn(c)
		}
	})
}

func (c *conn) addChannel(ch *channel) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.channels[ch] = true
}

func (c *conn) removeChannel(ch *channel) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.channels, ch)
}
