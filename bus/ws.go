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

type ServiceHandler interface {
	Call(e ebus.Event)
}

type ServiceHandleFunc func(e ebus.Event)

func (fn ServiceHandleFunc) Call(e ebus.Event) {
	fn(e)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type JoinedGroup struct {
	sync.Mutex

	groups map[string]bool // group id
}

func NewJoinedGroup() *JoinedGroup {
	return &JoinedGroup{
		groups: make(map[string]bool),
	}
}

func (g *JoinedGroup) Join(group_id string) {
	if g == nil {
		return
	}
	g.Lock()
	defer g.Unlock()

	g.groups[group_id] = true
}
func (g *JoinedGroup) Leave(group_id string) {
	if g == nil {
		return
	}
	g.Lock()
	defer g.Unlock()

	delete(g.groups, group_id)
}
func (g *JoinedGroup) All() []string {
	if g == nil {
		return nil
	}

	g.Lock()
	defer g.Unlock()

	all := make([]string, 0, len(g.groups))
	for id := range g.groups {
		all = append(all, id)
	}
	return all
}

type Agent struct {
	Notify      <-chan bool
	Box         *ebus.Bus
	JoinedGroup *JoinedGroup

	onclose func()
}

func NewAgent() *Agent {
	notify := make(chan bool, 1)
	agent := &Agent{
		Notify:      notify,
		Box:         ebus.New(),
		JoinedGroup: NewJoinedGroup(),

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

type Group map[string]bool

type Manager struct {
	sync.Mutex

	nextId int
	agents map[string]*Agent

	services map[string]ServiceHandler

	groups map[string]Group

	//analysis
	eventCount int
}

func NewManager() *Manager {
	return &Manager{
		nextId:   1,
		agents:   make(map[string]*Agent),
		services: make(map[string]ServiceHandler),
		groups:   make(map[string]Group),

		eventCount: 0,
	}
}

// method without @ prefix
func (m *Manager) Provide(method string, rpc ServiceHandler) {
	if strings.HasPrefix(method, "@") {
		panic(method)
	}

	m.Lock()
	defer m.Unlock()

	m.services[method] = rpc
}
func (m *Manager) ProvideFunc(method string, fn ServiceHandleFunc) {
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
		rpc := m.services[e.To[1:]]
		if rpc != nil {
			rpc.Call(e)
		}
	default:
		fmt.Println("discard:", e)
		m.eventCount--
	}
}

func (m *Manager) join(group_id, agent_id string) {
	agent := m.agents[agent_id]
	if agent == nil {
		return
	}

	group, ok := m.groups[group_id]
	if !ok {
		group = make(Group)
		m.groups[group_id] = group
	}

	group[agent_id] = true
	agent.JoinedGroup.Join(group_id)
}

func (m *Manager) leave(group_id, agent_id string) {
	agent := m.agents[agent_id]
	if agent == nil {
		return
	}
	agent.JoinedGroup.Leave(group_id)

	group, ok := m.groups[group_id]
	if !ok {
		return
	}

	delete(group, agent_id)
}

func (m *Manager) boardcast(group_id string, e ebus.Event) {
	group := m.groups[group_id]
	for agent_id := range group {
		e.To = agent_id
		m.emit(e)
	}
}

type RecordStore struct {
	sync.Mutex

	Data map[string]string
}

func NewRecordStore() *RecordStore {
	return &RecordStore{
		Data: make(map[string]string),
	}
}

func (rs *RecordStore) WithLock(fn func(rs *RecordStore)) {
	rs.Lock()
	defer rs.Unlock()
	fn(rs)
}

func serve() {
	mlog.TextMode()

	manager := NewManager()
	manager.ProvideFunc("manage", func(e ebus.Event) {
		fmt.Println(e)
		switch e.Topic {
		case "disconnected":
			for _, gid := range manager.agents[e.From].JoinedGroup.All() {
				manager.leave(gid, e.From)
			}
		}
	})
	manager.ProvideFunc("join", func(e ebus.Event) {
		group_id := e.Topic
		if group_id == "" {
			return
		}
		manager.join(group_id, e.From)
	})
	manager.ProvideFunc("leave", func(e ebus.Event) {
		group_id := e.Topic
		if group_id == "" {
			return
		}
		manager.leave(group_id, e.From)
	})
	manager.ProvideFunc("boardcast", func(e ebus.Event) {
		group_id := e.Topic
		if group_id == "" {
			return
		}
		e.From = "@boardcast"
		manager.boardcast(group_id, e)
	})

	// record
	rs := NewRecordStore()
	manager.ProvideFunc("record", func(e ebus.Event) {
		switch e.Topic {
		case "set":
			if len(e.Data) != 2 {
				return
			}
			k := e.Data[0]
			v := e.Data[1]
			needBoardcast := false
			rs.WithLock(func(rs *RecordStore) {
				old := rs.Data[k]
				if old != v {
					needBoardcast = true
				}
				rs.Data[k] = v
			})

			if needBoardcast {
				manager.boardcast("@record/"+k, ebus.Event{
					From:  "@record",
					Topic: "set",
					Data:  []string{k, v},
				})
			}
		case "get", "sync":
			if len(e.Data) != 1 {
				return
			}
			k := e.Data[0]
			v := ""
			rs.WithLock(func(rs *RecordStore) {
				v = rs.Data[k]
			})
			manager.emit(ebus.Event{
				From:  "@record",
				To:    e.From,
				Topic: "set",
				Data:  []string{k, v},
			})
			if e.Topic == "sync" {
				manager.join("@record/"+k, e.From)
			}
		}
	})

	manager.ProvideFunc("status", func(e ebus.Event) {
		switch e.Topic {
		case "agents":
			var agents []string
			for agent := range manager.agents {
				agents = append(agents, agent)
			}

			manager.emit(ebus.Event{
				From:  "@status",
				To:    e.From,
				Topic: e.Topic,
				Data:  agents,
			})
		case "event_count":
			manager.emit(ebus.Event{
				From:  "@status",
				To:    e.From,
				Topic: e.Topic,
				Data:  []string{fmt.Sprint(manager.eventCount)},
			})
		}
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

		manager.Emit(ebus.Event{
			From:  id,
			To:    "@manage",
			Topic: "connected",
		})
		defer manager.Emit(ebus.Event{
			From:  id,
			To:    "@manage",
			Topic: "disconnected",
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

	mlog.L().Info("serve " + ServeAddr)

	if ServeSSL == "on" {
		mlog.L().Info("open ssl")
		mlog.L().Error(http.ListenAndServeTLS(ServeAddr, SSLCert, SSLKey, nil))
	} else {
		mlog.L().Error(http.ListenAndServe(ServeAddr, nil))
	}
}
