package ebus

type Event struct {
	Topic string
	Data  string
}

func E(topic, data string) Event {
	return Event{
		Topic: topic,
		Data:  data,
	}
}
