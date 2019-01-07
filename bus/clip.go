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

	c := ebus.NewClient("ws://39.105.42.45:8100/", func(e ebus.Event) {
		//c := NewClient("ws://localhost:8100/", func(e ebus.Event) {
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

func say(s string) {
	exec.Command("/Users/yangjingrong/dev/bin/push", s).Run()
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
