package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/immofon/ebus"
	"github.com/immofon/mlog"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Agent struct {
	Notify <-chan bool

	Box *ebus.Bus
}

func NewAgent() *Agent {
	notify := make(chan bool, 1)
	agent := &Agent{
		Notify: notify,

		Box: ebus.New(),
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
}

func NewManager() *Manager {
	return &Manager{
		nextId: 1,
		agents: make(map[string]*Agent),
	}
}

func (m *Manager) AddAgent(a *Agent) string {
	m.Lock()
	defer m.Unlock()

	id := fmt.Sprintf("#%d", m.nextId)
	m.nextId++

	m.agents[id] = a
	return id
}

func (m *Manager) Remove(id string) {
	m.Lock()
	defer m.Unlock()

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

func main() {
	mlog.TextMode()

	manager := NewManager()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		agent := NewAgent()
		id := manager.AddAgent(agent)
		defer manager.Remove(id)

		agent.Box.Emit(ebus.E("id", id))

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
							if e.Topic == "" {
								return
							}

							if err := conn.WriteJSON(e); err == nil {
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
			if err := conn.ReadJSON(&e); err != nil {
				return
			}

			manager.Get(e.Topic).Emit(e)
		}

	})

	mlog.L().Info("serve :8100")
	http.ListenAndServe("localhost:8100", nil)
}
