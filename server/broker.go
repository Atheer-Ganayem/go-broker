package broker

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"sync"
)

type broker struct {
	addr string

	conns   map[*conn]bool
	connsMu sync.RWMutex

	chByName map[string]*channel
	chByID   map[uint16]*channel
	chMu     sync.RWMutex
}

func NewBroker(addr string) *broker {
	return &broker{
		addr:     addr,
		conns:    make(map[*conn]bool),
		chByName: make(map[string]*channel),
		chByID:   make(map[uint16]*channel),
	}
}

func (b *broker) Start() error {
	l, err := net.Listen("tcp", b.addr)
	if err != nil {
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf("faild to accept conn: %s\n", err)
			continue
		}

		c := newConn(conn, b)
		b.connect(c)

		go c.listener()
		go c.readLoop()
	}
}

func (b *broker) connect(c *conn) {
	b.connsMu.Lock()
	defer b.connsMu.Unlock()

	b.conns[c] = true
}

func (b *broker) disconnect(c *conn) {
	b.connsMu.Lock()

	exists := b.conns[c]
	if !exists {
		return
	}

	delete(c.broker.conns, c)
	b.connsMu.Unlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	for ch, _ := range c.channels {
		ch.removeConn(c)
	}
}

func (b *broker) getChannelByName(name string) *channel {
	b.chMu.RLock()
	defer b.chMu.RUnlock()

	return b.chByName[name]
}

func (b *broker) getChannelByID(id uint16) *channel {
	b.chMu.RLock()
	defer b.chMu.RUnlock()

	return b.chByID[id]
}

// Tries to get the channel, if not exists it creates one and returns it.
func (b *broker) getAddChannel(name string) *channel {
	b.chMu.Lock()
	defer b.chMu.Unlock()

	// check if already exists
	if ch := b.chByName[name]; ch != nil {
		return ch
	}

	ch := b.unsafeAddChannel(name)
	return ch
}

func (b *broker) addChannel(name string) *channel {
	b.chMu.Lock()
	defer b.chMu.Unlock()

	ch := b.unsafeAddChannel(name)
	return ch
}

// without locks
func (b *broker) unsafeAddChannel(name string) *channel {
	// create new channel and generate an unused ID
	ch := newChannel(name, uint16(rand.Intn(math.MaxUint16)))
	for b.chByID[ch.id] != nil {
		ch.id = uint16(rand.Intn(math.MaxUint16))
	}

	//add channel to the maps
	b.chByName[name] = ch
	b.chByID[ch.id] = ch

	return ch
}

func (b *broker) subscribe(c *conn, chName string) *channel {
	ch := b.getAddChannel(chName)

	ch.addConn(c)
	c.addChannel(ch)

	return ch
}

func (b *broker) unsubscribe(c *conn, id uint16) {
	ch := b.getChannelByID(id)

	c.removeChannel(ch)
	ch.removeConn(c)
}
