package main

import (
	"bytes"
	"os/exec"
	"time"

	"github.com/immofon/ebus"
)

func clip() {
	c := NewClient("ws://localhost:8100/", func(e ebus.Event) {
	})

	ch := make(chan string)
	go generate(ch)

	cache := ""
	for s := range ch {
		if s != cache {
			c.Emit(ebus.Event{
				To:    "#clip",
				Topic: "#clip",
				Data:  []string{s},
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
