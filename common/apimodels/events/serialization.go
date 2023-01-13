package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	newMessagePrefix    = []byte("NewMessage|")
	systemMessagePrefix = []byte("SystemMessage|")
)

func Serialize(event any) ([]byte, error) {
	var prefix []byte
	switch event.(type) {
	case NewMessage:
		prefix = newMessagePrefix
	case SystemMessage:
		prefix = systemMessagePrefix
	default:
		return nil, fmt.Errorf("unknown event type '%T'", event)
	}

	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	return append(prefix, jsonBytes...), nil
}

func Parse(messageBytes []byte) (any, error) {
	pipePos := bytes.IndexRune(messageBytes, '|')
	if pipePos <= 0 || pipePos == len(messageBytes)-1 {
		return nil, errors.New("provided bytes slice is not a valid event representation")
	}

	prefix := messageBytes[:pipePos+1]
	jsonBytes := messageBytes[pipePos+1:]

	var event any
	var err error
	// currently Unmarshal() can't figure out underlying type of event
	// may be changed later with https://github.com/golang/go/issues/26946
	switch {
	case bytes.Equal(prefix, newMessagePrefix):
		nm := NewMessage{}
		err = json.Unmarshal(jsonBytes, &nm)
		event = nm
	case bytes.Equal(prefix, systemMessagePrefix):
		sm := SystemMessage{}
		err = json.Unmarshal(jsonBytes, &sm)
		event = sm
	default:
		return nil, fmt.Errorf("unexpected event prefix '%v'", prefix)
	}

	return event, err
}
