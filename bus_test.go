package ebus

import (
	"fmt"
	"testing"
	"time"
)

func Test_Bus(t *testing.T) {
	bus := New()
	bus.SetAfterEmit(func(e Event) {
		fmt.Println(e.Topic, ":", e.Topic)
	})

	go func(bus *Bus) {
		for range time.NewTicker(time.Millisecond * 10).C {
			e := bus.Get()
			if e.Topic != "" {
				fmt.Println("Handle:", e.Topic, ":", e.Topic)
				bus.Delete()
			}
		}
	}(bus)

	bus.Emit(E("print", "hello world"))
	bus.Emit(E("hello", "world"))
	bus.Emit(E("message", "push"))

	time.Sleep(time.Millisecond * 500)
}
