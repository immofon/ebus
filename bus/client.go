package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/immofon/ebus"
)

func client() {
	c := ebus.NewClient("ws://39.105.42.45:8100/", func(e ebus.Event) {
		//c := NewClient("ws://localhost:8100/", func(e ebus.Event) {
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
