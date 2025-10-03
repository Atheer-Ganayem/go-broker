package client

import "github.com/Atheer-Ganayem/go-broker/internal"

type channel struct {
	id   uint16
	name string

	messages chan *internal.Message
}
