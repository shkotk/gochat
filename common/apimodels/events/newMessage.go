package events

import "time"

type NewMessage struct {
	Producer string
	Time     time.Time
	Text     string
}

func (m NewMessage) GetProducer() string          { return m.Producer }
func (m *NewMessage) SetProducer(producer string) { m.Producer = producer }
func (m NewMessage) GetTime() time.Time           { return m.Time }
func (m *NewMessage) SetTime(time time.Time)      { m.Time = time }
