package events

import "time"

type NewMessage struct {
	Producer string
	Time     time.Time
	Text     string
}
