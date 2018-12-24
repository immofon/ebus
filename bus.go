package ebus

import (
	"container/list"
	"sync"
)

type Bus struct {
	sync.Mutex
	events    *list.List
	afterEmit func(Event)
}

func New() *Bus {
	return &Bus{
		events:    list.New(),
		afterEmit: func(Event) {},
	}
}

// Call AfterEmit after unlock, it's out of the lock.
func (b *Bus) Emit(e Event) {
	if b == nil {
		return
	}

	defer b.afterEmit(e)

	b.Lock()
	defer b.Unlock()

	b.events.PushBack(e)
}

func (b *Bus) Get() Event {
	b.Lock()
	defer b.Unlock()
	elem := b.events.Front()
	if elem != nil {
		return elem.Value.(Event)
	} else {
		return Event{}
	}
}

func (b *Bus) Delete() {
	b.Lock()
	defer b.Unlock()

	elem := b.events.Front()
	if elem != nil {
		b.events.Remove(elem)
	}
}

func (b *Bus) SetAfterEmit(on func(Event)) {
	b.Lock()
	defer b.Unlock()

	if on == nil {
		on = func(Event) {}
	}

	b.afterEmit = on
}
