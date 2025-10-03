package internal

import (
	"bufio"
	"encoding/binary"
	"math"
	"net"
	"slices"
)

const (
	MaxMessageSize   = 1024 * 512 // 1/2 MB
	bufSize          = 1024 * 4   // 4KB
	MessagesChanSize = 1024
	Space            = 0x20
)

// OPCODES (op names)
type opcode byte

const (
	OpcodeClose opcode = iota << 3
	OpcodeErr
	OpcodeSub
	OpcodeUnsub
	OpcodePing
	OpcodePong
	OpcodePub
	OpcodeMsg
	OpcodeACK
	OpcodeInfoChID
)

var validOpcodes = []opcode{OpcodeClose, OpcodeErr, OpcodeSub, OpcodeUnsub, OpcodePing, OpcodePong, OpcodePub, OpcodeMsg, OpcodeACK, OpcodeInfoChID}

/*
PROTOCOL:
	[5 bits opcode | messageID exists? | RSV1 | RSV2]
	[big-endian uint16 channelID]
	[big-endian uint64 messageID (If messageID set to 1)]
	[payload length: 3 cases:
		1) val < 254 => val
		2) val = 254 => next 2 bytes are the payload length (uint16)
		3) val = 255 => next 4 bytes are the payload length (uint32)
	]
	[payload]

  opClose: opcode + payloadLen + payload
	opErr: opcode + payloadLen  + payload
	subscribe: opcode + payloaLen (channel name length) + payload (channel-name)
	unsubscribe: opcode + channelID
	ping: opcode
	pong: opcode
	publish: opcode + channelID + payloadLen (4 bytes) + payload
	opMsg: opcode + channelID + [msgID?] + payloadLen (4 bytes) + payload
	ack: opcode + channelID + messageID
	opInfoChID: opcode + channelID + payloaLen (channel name length) + payload (channel-name)

*/

func isValidOpcode(opcode opcode) bool {
	return slices.Contains(validOpcodes, opcode)
}

type ConnReader struct {
	br *bufio.Reader
}

func NewConnReader(c net.Conn) *ConnReader {
	return &ConnReader{bufio.NewReaderSize(c, bufSize)}
}

func (r *ConnReader) Buffer() *bufio.Reader {
	return r.br
}

func (r *ConnReader) PeekDiscard(n int) ([]byte, error) {
	b, err := r.br.Peek(n)
	if err != nil {
		return nil, err
	}

	// will always succeed
	_, _ = r.br.Discard(n)

	return b, nil
}

// return: opcode, messageID exists, rsv1, rsv2, error
func (r *ConnReader) ReadOpcode() (op opcode, existsMsgID bool, rsv1 bool, rsv2 bool, err error) {
	b, err := r.PeekDiscard(1)
	if err != nil {
		return
	}

	op = opcode(b[0] & 0b11111000)
	if !isValidOpcode(op) {
		err = errInvalidOpocde
		return
	}

	existsMsgID = b[0]&0b00000100 == 1
	rsv1 = b[0]&0b00000001 == 1
	rsv2 = b[0]&0b00000010 == 1

	if rsv1 || rsv2 {
		err = errRSVBitsSet
		return
	}

	return
}

func (r *ConnReader) ReadChannelID() (uint16, error) {
	b, err := r.PeekDiscard(2)
	if err != nil {
		return 0, err
	}

	id := binary.BigEndian.Uint16(b)

	return id, nil
}

func (r *ConnReader) ReadMessageID() (uint64, error) {
	b, err := r.PeekDiscard(8)
	if err != nil {
		return 0, err
	}

	id := binary.BigEndian.Uint64(b)

	return id, nil
}

func (r *ConnReader) ReadPayloadLength() (uint, error) {
	b, err := r.PeekDiscard(1)
	if err != nil {
		return 0, err
	}

	var len uint

	switch b[0] {
	case 254:
		b, err := r.PeekDiscard(2)
		if err != nil {
			return 0, err
		}
		len = uint(binary.BigEndian.Uint16(b))
	case 255:
		b, err := r.PeekDiscard(4)
		if err != nil {
			return 0, err
		}
		len = uint(binary.BigEndian.Uint32(b))
	default:
		len = uint(b[0])
	}

	if len > MaxMessageSize {
		// handle more
		return len, errPayloadTooBig
	}

	return len, nil
}

func EncodeMessage(msg *Message) []byte {
	size := 1
	var appendChID, appendMsgID, appendPayload bool

	if msg.opcode == OpcodeUnsub || msg.opcode == OpcodePub || msg.opcode == OpcodeMsg || msg.opcode == OpcodeACK || msg.opcode == OpcodeInfoChID {
		size += 2
		appendChID = true
	}
	if (msg.opcode == OpcodeMsg || msg.opcode == OpcodeACK) && msg.ID != 0 {
		size += 8
		appendMsgID = true
	}
	if msg.opcode == OpcodeSub || msg.opcode == OpcodeClose || msg.opcode == OpcodeErr || msg.opcode == OpcodePub || msg.opcode == OpcodeMsg || msg.opcode == OpcodeInfoChID {
		appendPayload = true
		size = size + 1 + len(msg.payload)

		if len(msg.payload) > 253 && len(msg.payload) <= math.MaxUint16 {
			size += 2
		} else if len(msg.payload) > 253 && len(msg.payload) <= math.MaxUint32 {
			size += 4 // payload len
		} else {
			// TODO: MUST RETURN AN ERR
		}
	}

	b := make([]byte, 0, size)
	b = append(b, byte(msg.opcode))
	if appendMsgID {
		b[0] |= 1 << 2
	}

	if appendChID {
		b = binary.BigEndian.AppendUint16(b, msg.ChannelID)
	}

	if appendMsgID {
		b = binary.BigEndian.AppendUint64(b, msg.ID)
	}

	if appendPayload {
		if len(msg.payload) <= 253 {
			b = append(b, byte(len(msg.payload)))
		} else if len(msg.payload) <= math.MaxUint16 {
			b = binary.BigEndian.AppendUint16(b, uint16(len(msg.payload)))
		} else if len(msg.payload) <= math.MaxUint32 {
			b = binary.BigEndian.AppendUint32(b, uint32(len(msg.payload)))
		}
		b = append(b, msg.payload...)
	}

	return b
}
