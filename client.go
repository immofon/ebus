package ebus

import (
	"github.com/gorilla/websocket"
)

type updateClientFn func(*Client)

type Client struct {
	ch chan updateClientFn

	conn *websocket.Conn
}

// panic: dial websocket
func NewClient(url string, emit func(Event)) *Client {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		panic(err)
	}
	c := &Client{
		ch: make(chan updateClientFn),

		conn: conn,
	}
	go c.Serve(emit)

	return c
}

func (c *Client) Serve(emit func(Event)) {
	go func() {
		defer close(c.ch)
		defer c.conn.Close()

		var e Event
		for {
			_, raw, err := c.conn.ReadMessage()
			if err != nil {
				return
			}
			e = e.ClientUnmarshal(string(raw))

			emit(e)
		}
	}()

	for fn := range c.ch {
		fn(c)
	}
}

func (c *Client) Emit(e Event) {
	c.ch <- func(c *Client) {
		c.conn.WriteMessage(websocket.TextMessage, []byte(e.ClientMarshal()))
	}
}
