package main

import (
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/immofon/ebus"
)

type updateClientFn func(*Client)

type Client struct {
	ch chan updateClientFn

	conn *websocket.Conn
}

// panic: dial websocket
func NewClient(url string) *Client {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		panic(err)
	}
	return &Client{
		ch: make(chan updateClientFn),

		conn: conn,
	}
}

func (c *Client) Serve() {
	go func() {
		defer close(c.ch)
		defer c.conn.Close()

		var e ebus.Event
		for {
			_, raw, err := c.conn.ReadMessage()
			if err != nil {
				return
			}
			e = e.ClientUnmarshal(string(raw))

			fmt.Println(e.From, ":[", e.Topic, "]:", e.Data)
		}
	}()

	for fn := range c.ch {
		fn(c)
	}
}

func (c *Client) Emit(e ebus.Event) {
	c.ch <- func(c *Client) {
		c.conn.WriteMessage(websocket.TextMessage, []byte(e.ClientMarshal()))
	}
}

func client() {
	c := NewClient("ws://localhost:8100/")
	go c.Serve()

	var to, data string
	for {
		fmt.Scanf("%s %s", &to, &data)
		c.Emit(ebus.E(to, "chat", data))
	}
}
