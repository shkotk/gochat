package events

import "time"

type SystemMessage struct {
	Text string
	Time time.Time
}

func (m SystemMessage) GetTime() time.Time      { return m.Time }
func (m *SystemMessage) SetTime(time time.Time) { m.Time = time }
