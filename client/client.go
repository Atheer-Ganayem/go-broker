package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Atheer-Ganayem/go-broker/internal"
)

type Client struct {
	conn      net.Conn
	reader    *internal.ConnReader
	closed    chan struct{}
	closeOnce sync.Once

	in  chan *internal.Message
	out chan *internal.Message

	chByName        map[string]*channel
	chByID          map[uint16]*channel
	waitingChannels map[string]bool
	mu              sync.RWMutex
}

func NewClient(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	c := &Client{
		conn:   conn,
		reader: internal.NewConnReader(conn),
		closed: make(chan struct{}),
		out:    make(chan *internal.Message, internal.MessagesChanSize),
	}

	return c, nil
}

func (c *Client) Listen() {
	go c.readLoop()
	go c.writeLoop()
	<-c.closed
}

func (c *Client) readLoop() {
	for {
		op, mID, _, _, err := c.reader.ReadOpcode()
		if err != nil {
			// handle
			return
		}

		switch op {
		case internal.OpcodeClose:
			// close
			return
		case internal.OpcodePing:
			err = c.pong()
		case internal.OpcodePong:
			err = c.conn.SetReadDeadline(time.Now().Add(time.Minute / 2))
		case internal.OpcodeMsg:
			err = c.handleMsg(mID)
		case internal.OpcodeInfoChID:
			err = c.handleChInfo()

		default:
			return
			// close conn
		}

		if err != nil {
			fmt.Println(err)
			// handle
			return
		}
	}
}

func (c *Client) pong() error {
	err := c.conn.SetReadDeadline(time.Now().Add(time.Minute / 2))
	if err != nil {
		return err
	}

	msg := internal.NewMessageEmpty(internal.OpcodePong)
	select {
	case c.out <- msg:
	default:
		fmt.Println("channel full")
	}

	return nil
}

func (c *Client) handleMsg(mID bool) error {
	chID, err := c.reader.ReadChannelID()
	if err != nil {
		return err
	}

	// i should store channels that im connected to and valiudate chID

	var msgID uint64
	if mID {
		msgID, err = c.reader.ReadMessageID()
		if err != nil {
			return err
		}
	}

	len, err := c.reader.ReadPayloadLength()
	if err != nil {
		return err
	}

	msg, err := internal.NewMessage(internal.OpcodeMsg, len, c.reader.Buffer())
	if err != nil {
		return err
	}
	msg.ID = msgID

	select {
	case c.in <- msg:
	default:
		fmt.Println("in channel full")
	}

	fmt.Println(chID, msg)
	// send message to ch

	return nil
}

func (c *Client) handleChInfo() error {
	id, err := c.reader.ReadChannelID()
	if err != nil {
		return err
	}

	len, err := c.reader.ReadPayloadLength()
	if err != nil {
		return err
	}

	chName, err := c.reader.PeekDiscard(int(len))
	if err != nil {
		return err
	}

	fmt.Println(string(chName), ":", id)

	return nil
}

func (c *Client) writeLoop() {
	for msg := range c.out {
		b := internal.EncodeMessage(msg)
		c.conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
		_, err := c.conn.Write(b)
		if err != nil {
			panic(fmt.Errorf("client write failed: %w", err))
		}
	}
}

func (c *Client) Sub(chName string) {
	msg := internal.NewMessageSize(internal.OpcodeSub, uint(len(chName)))
	_, _ = msg.Write([]byte(chName))
	// msg.payload = append(msg.payload, []byte(chName)...)

	c.mu.Lock()
	// if c.chByName[chName] !=
	c.mu.Unlock()

	select {
	case c.out <- msg:
	default:
		fmt.Println("c.out full")
	}
}
