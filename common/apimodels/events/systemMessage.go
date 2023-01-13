package events

import "time"

type SystemMessage struct {
	Text string
	Time time.Time
}
