package events

import "time"

type Produced interface {
	GetProducer() string
	SetProducer(string)
}

type Timed interface {
	GetTime() time.Time
	SetTime(time.Time)
}
