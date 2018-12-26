package ebus

import "strings"

type Event struct {
	From  string
	To    string
	Topic string
	Data  []string
}

func E(to, topic string, data ...string) Event {
	return Event{
		From:  "",
		To:    to,
		Topic: topic,
		Data:  data,
	}
}

func marshal(target, topic string, data []string) string {
	return strings.Join([]string{target, topic, strings.Join(data, "\x1f")}, "\x1f")
}

func unmarshal(raw string) (target, topic string, data []string) {
	rawsp := strings.Split(raw, "\x1f")
	target = rawsp[0]
	topic = rawsp[1]
	data = rawsp[2:]
	return
}

func (e Event) ServerMarshal() string {
	return marshal(e.From, e.Topic, e.Data)
}
func (e Event) ClientUnmarshal(raw string) Event {
	target, topic, data := unmarshal(raw)
	return Event{
		From:  target,
		Topic: topic,
		Data:  data,
	}
}

func (e Event) ClientMarshal() string {
	return marshal(e.To, e.Topic, e.Data)
}

func (e Event) ServerUnmarshal(raw string) Event {
	target, topic, data := unmarshal(raw)
	return Event{
		To:    target,
		Topic: topic,
		Data:  data,
	}
}
