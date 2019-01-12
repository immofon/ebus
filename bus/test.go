package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/immofon/ebus"
)

func test() {
	defer func() {
		recover()
	}()

	countCh := make(chan int)

	go func(counts <-chan int) {
		ticker := time.NewTicker(time.Second)
		lastCount := 0
		thisCount := 0

		for {
			select {
			case c := <-counts:
				thisCount = c
			case <-ticker.C:
				if lastCount == 0 {
					lastCount = thisCount
					continue
				}

				count_persec := thisCount - lastCount
				lastCount = thisCount

				fmt.Println(count_persec, "/sec")

			}
		}
		for c := range counts {
			if lastCount == 0 {
				lastCount = c
				continue
			}

		}
	}(countCh)

	c := ebus.NewClient(ServerURL, func(e ebus.Event) {
		if e.From == "@status" && e.Topic == "event_count" {
			event_count, _ := strconv.Atoi(e.Data[0])
			countCh <- event_count
		}
	})

	for {
		c.Emit(ebus.Event{
			To:    "@status",
			Topic: "event_count",
		})
	}
}
