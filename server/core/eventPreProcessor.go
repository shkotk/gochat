package core

import (
	"fmt"
	"time"

	"github.com/shkotk/gochat/common/apimodels/events"
	"github.com/shkotk/gochat/server/interfaces"
)

type EventPreProcessor struct{}

func NewEventPreProcessor() *EventPreProcessor { return &EventPreProcessor{} }

func (p *EventPreProcessor) PreProcess(event any, producer interfaces.Client) error {
	// filter expected incoming event types
	switch event.(type) {
	case *events.NewMessage:
	default:
		return fmt.Errorf("chat: got event of unexpected type %T from client '%s'",
			event, producer.ID())
	}

	if event, ok := event.(events.Produced); ok {
		event.SetProducer(producer.ID())
	}
	if event, ok := event.(events.Timed); ok {
		event.SetTime(time.Now())
	}

	return nil
}
