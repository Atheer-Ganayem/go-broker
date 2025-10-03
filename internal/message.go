package internal

import "io"

type Message struct {
	ID        uint64
	opcode    opcode
	ChannelID uint16
	payload   []byte

	channelName string // for opSub
}

func NewMessageEmpty(opcode opcode) *Message {
	return &Message{
		opcode: opcode,
	}
}

func NewMessageSize(opcode opcode, size uint) *Message {
	return &Message{
		opcode:  opcode,
		payload: make([]byte, 0, size),
	}
}

func NewMessage(opcode opcode, size uint, r io.Reader) (*Message, error) {
	m := NewMessageSize(opcode, size)

	m.payload = m.payload[:size]
	n, err := r.Read(m.payload)
	if err != nil {
		return nil, err
	} else if n != int(size) {
		return nil, errIncompleteRead
	}

	return m, nil
}

func (m *Message) Write(b []byte) (int, error) {
	offset := len(m.payload)
	space := cap(m.payload) - offset
	if space == 0 {
		return 0, errPayloadFull
	}

	n := min(len(b), space)

	m.payload = m.payload[:offset+n]

	copy(m.payload[offset:offset+n], b[:n])

	return n, nil
}
