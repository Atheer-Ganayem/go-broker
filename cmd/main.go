package main

import (
	"time"

	client "github.com/Atheer-Ganayem/go-broker/client"
	broker "github.com/Atheer-Ganayem/go-broker/server"
)

func main() {
	b := broker.NewBroker(":8080")
	go func() {
		if err := b.Start(); err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Second / 2)

	c, err := client.NewClient(":8080")
	if err != nil {
		panic(err)
	}

	go tests(c)

	c.Listen()
}

func tests(c *client.Client) {
	time.Sleep(time.Second / 2)
	c.Sub("hello")
}
