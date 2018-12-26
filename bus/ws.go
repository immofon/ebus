package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/immofon/ebus"
	"github.com/immofon/mlog"
)

type RPCHandler interface {
	Call(e ebus.Event, emit func(ebus.Event))
}

type RPCHandleFunc func(e ebus.Event, emit func(ebus.Event))

func (fn RPCHandleFunc) Call(e ebus.Event, emit func(ebus.Event)) {
	fn(e, emit)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Agent struct {
	Notify <-chan bool
	Box    *ebus.Bus

	onclose func()
}

func NewAgent() *Agent {
	notify := make(chan bool, 1)
	agent := &Agent{
		Notify: notify,
		Box:    ebus.New(),

		onclose: func() {
			close(notify)
		},
	}

	agent.Box.SetAfterEmit(func(_ ebus.Event) {
		select {
		case notify <- true:
		default:
		}
	})

	return agent
}

type Manager struct {
	sync.Mutex

	nextId int
	agents map[string]*Agent
	rpcs   map[string]RPCHandler

	//analysis
	eventCount int
}

func NewManager() *Manager {
	return &Manager{
		nextId: 1,
		agents: make(map[string]*Agent),
		rpcs:   make(map[string]RPCHandler),
	}
}

// method without @ prefix
func (m *Manager) Provide(method string, rpc RPCHandler) {
	if strings.HasPrefix(method, "@") {
		panic(method)
	}

	m.Lock()
	defer m.Unlock()

	m.rpcs[method] = rpc
}
func (m *Manager) ProvideFunc(method string, fn RPCHandleFunc) {
	m.Provide(method, fn)
}

func (m *Manager) AddAgent(a *Agent) string {
	m.Lock()
	defer m.Unlock()

	id := fmt.Sprintf("&%d", m.nextId)
	m.nextId++

	m.agents[id] = a
	return id
}

func (m *Manager) Remove(id string) {
	m.Lock()
	defer m.Unlock()

	agent := m.agents[id]
	if agent == nil {
		return
	}
	agent.Box.SetAfterEmit(nil)
	agent.onclose()
	delete(m.agents, id)
}

func (m *Manager) Get(id string) *ebus.Bus {
	m.Lock()
	defer m.Unlock()

	ag := m.agents[id]
	if ag == nil {
		return nil
	}
	return ag.Box
}

func (m *Manager) Emit(e ebus.Event) {
	m.Lock()
	defer m.Unlock()

	m.emit(e)
}

func (m *Manager) emit(e ebus.Event) {
	m.eventCount++
	switch e.To[0] {
	case '&':
		agent := m.agents[e.To]
		if agent != nil {
			agent.Box.Emit(e)
		}
	case '@':
		rpc := m.rpcs[e.To[1:]]
		if rpc != nil {
			rpc.Call(e, m.emit)
		}
	default:
		fmt.Println("discard:", e)
		m.eventCount--
	}
}

func serve() {
	mlog.TextMode()

	tasks := make(chan func(), 100)
	go func() {
		for task := range tasks {
			task()
		}
	}()
	channels := make(map[string](map[string]bool))
	join := func(cid, id string) {
		tasks <- func() {
			channel, ok := channels[cid]
			if !ok {
				channel = make(map[string]bool)
				channels[cid] = channel
			}

			channel[id] = true
		}
	}
	leave := func(cid, id string) {
		tasks <- func() {
			channel, ok := channels[cid]
			if ok {
				delete(channel, id)
				if len(channel) == 0 {
					delete(channels, cid)
				}
			}
		}
	}
	getChannel := func(cid string) map[string]bool {
		ch := make(chan map[string]bool)
		defer close(ch)
		tasks <- func() {
			ch <- channels[cid]
		}
		return <-ch
	}

	manager := NewManager()
	manager.ProvideFunc("join", func(e ebus.Event, emit func(ebus.Event)) {
		join(e.Topic, e.From)
	})
	manager.ProvideFunc("leave", func(e ebus.Event, emit func(ebus.Event)) {
		leave(e.Topic, e.From)
	})
	manager.ProvideFunc("pub", func(e ebus.Event, emit func(ebus.Event)) {
		for id := range getChannel(e.Topic) {
			emit(ebus.Event{
				From:  "@pub",
				To:    id,
				Topic: e.Topic,
				Data:  e.Data,
			})
		}
	})
	manager.ProvideFunc("analysis", func(e ebus.Event, emit func(ebus.Event)) {
		emit(ebus.Event{
			From:  "@analysis",
			To:    e.From,
			Topic: "event_count",
			Data:  []string{fmt.Sprint(manager.eventCount)},
		})
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		agent := NewAgent()
		id := manager.AddAgent(agent)
		defer manager.Remove(id)

		agent.Box.Emit(ebus.Event{
			From:  "@manager",
			Topic: "@set.id",
			Data:  []string{id},
		})

		// send loop
		go func() {
			for {
				select {
				case _, ok := <-agent.Notify:
					if !ok {
						return
					}

					func() {
						for {
							e := agent.Box.Get()
							if e.From == "" || e.Topic == "" {
								return
							}

							if err := conn.WriteMessage(websocket.TextMessage, []byte(e.ServerMarshal())); err == nil {
								agent.Box.Delete()
							}
						}
					}()

				}
			}
		}()

		// recv loop
		var e ebus.Event
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			e = e.ServerUnmarshal(string(raw))

			e.From = id
			manager.Emit(e)
		}

	})

	mlog.L().Info("serve :8100")
	http.ListenAndServe("localhost:8100", nil)
}
