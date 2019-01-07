package main

import (
	"bytes"
	"os/exec"
	"time"

	"github.com/immofon/ebus"
)

func clip() {
	defer func() {
		recover()
	}()

	c := ebus.NewClient(ServerURL, func(e ebus.Event) {
	})

	ch := make(chan string)
	go generate(ch)

	cache := ""
	for s := range ch {
		if s != cache {
			c.Emit(ebus.Event{
				To:    "@record",
				Topic: "set",
				Data:  []string{"clip", s},
			})
			cache = s
		}
	}
}

func generate(ch chan<- string) {
	buf := bytes.NewBuffer(nil)
	for {
		buf.Reset()
		cmd := exec.Command("pbpaste")
		cmd.Stdout = buf
		cmd.Run()

		ch <- buf.String()
		time.Sleep(time.Millisecond * 100)
	}
}
