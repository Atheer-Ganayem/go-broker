package broker

import (
	"errors"
	"fmt"
	"time"

	"github.com/Atheer-Ganayem/go-broker/internal"
)

func (c *conn) readLoop() {
	defer c.netConn.Close()
	for {
		op, existMsgID, _, _, err := c.reader.ReadOpcode()
		if err != nil {
			// handle
			return
		}

		fmt.Println("conn read loop - existMsgID:", existMsgID)

		switch op {
		case internal.OpcodeClose:
			// close
			return
		case internal.OpcodeSub:
			err = c.handleSub()
		case internal.OpcodeUnsub:
			err = c.handleUnsub()
		case internal.OpcodePing:
			err = c.pong()
		case internal.OpcodePong:
			err = c.netConn.SetReadDeadline(time.Now().Add(time.Minute / 2))
		case internal.OpcodePub:
			err = c.handlePub()
		case internal.OpcodeACK:
			err = c.handleAck()

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

func (c *conn) handleSub() error {
	len, err := c.reader.ReadPayloadLength()
	if err != nil {
		return err
	}

	if len == 0 {
		return errors.New("invalid channel name length")
	}

	rawChName, err := c.reader.PeekDiscard(int(len))
	if err != nil {
		return err
	}

	ch := c.broker.subscribe(c, string(rawChName))

	// respond with channel id
	msg := internal.NewMessageSize(internal.OpcodeInfoChID, len)
	msg.ChannelID = ch.id
	_, _ = msg.Write(rawChName)

	select {
	case c.messages <- msg:
	default:
		fmt.Println("chan full bro. FULL!")
	}

	return nil
}

func (c *conn) handleUnsub() error {
	id, err := c.reader.ReadChannelID()
	if err != nil {
		return err
	}

	c.broker.unsubscribe(c, id)

	return nil
}

func (c *conn) pong() error {
	err := c.netConn.SetReadDeadline(time.Now().Add(time.Minute / 2))
	if err != nil {
		return err
	}

	msg := internal.NewMessageEmpty(internal.OpcodePong)
	select {
	case c.messages <- msg:
	default:
		fmt.Println("channel full")
	}

	return nil
}

func (c *conn) handlePub() error {
	ch, err := c.readChannelID()
	if err != nil {
		return err
	}

	len, err := c.reader.ReadPayloadLength()
	if err != nil {
		return err
	}

	if len > internal.MaxMessageSize {
		// handle more
		return errors.New("message too long")
	}

	msg, err := internal.NewMessage(internal.OpcodeMsg, len, c.reader.Buffer())
	if err != nil {
		return err
	}
	msg.ChannelID = ch.id

	select {
	case ch.messages <- msg:
	default:
		fmt.Println("message channel full")
	}

	return nil
}

func (c *conn) handleAck() error {
	ch, err := c.readChannelID()
	if err != nil {
		return err
	}

	msgID, err := c.reader.ReadMessageID()
	if err != nil {
		return err
	}

	fmt.Printf("ack: %s, %s", ch.name, msgID)
	// send message to ch

	return nil
}

// #############################
// ########## HELPERS ##########
// ##############################

func (c *conn) readChannelID() (*channel, error) {
	id, err := c.reader.ReadChannelID()
	if err != nil {
		return nil, err
	}

	ch := c.broker.getChannelByID(id)
	if ch == nil {
		return nil, errors.New("channel doesnt exist")
	}

	return ch, nil
}
