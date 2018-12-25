package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/immofon/ebus"
)

type updateClientFn func(*Client)

type Client struct {
	ch chan updateClientFn

	conn *websocket.Conn
}

// panic: dial websocket
func NewClient(url string, emit func(ebus.Event)) *Client {
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

func (c *Client) Serve(emit func(ebus.Event)) {
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

			emit(e)
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
	c := NewClient("ws://localhost:8100/", func(e ebus.Event) {
		fmt.Println(e.From, ":[", e.Topic, "]:", e.Data)
	})

	help()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		raw := strings.Split(scanner.Text(), " ") // Println will add back the final '\n'
		if len(raw) >= 2 {
			c.Emit(ebus.Event{
				To:    raw[0],
				Topic: raw[1],
				Data:  raw[2:],
			})
		} else {
			help()
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

func help() {
	fmt.Println("to topic [data1 data2 ...")
}
