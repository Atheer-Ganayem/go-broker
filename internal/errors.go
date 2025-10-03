package internal

import "errors"

var (
	errInvalidOpocde  = errors.New("received invalid opocde")
	errRSVBitsSet     = errors.New("RSV bits set (expected them not to be)")
	errPayloadTooBig  = errors.New("payload too big")
	errIncompleteRead = errors.New("failed to read full payload")
	errPayloadFull    = errors.New("couldn't write to payload. payload full")
)
