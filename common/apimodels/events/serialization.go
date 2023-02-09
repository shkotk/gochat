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

// Serializes event to a JSON string with prefix representing event type.
// A pointer to an event struct of a known type is expected.
func Serialize(event any) ([]byte, error) {
	var prefix []byte
	switch event.(type) {
	case *NewMessage:
		prefix = newMessagePrefix
	case *SystemMessage:
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

// Parses event from a JSON string with prefix representing event type.
// A pointer to an event struct is returned.
func Parse(messageBytes []byte) (any, error) {
	pipePos := bytes.IndexRune(messageBytes, '|')
	if pipePos <= 0 || pipePos == len(messageBytes)-1 {
		return nil, errors.New("provided bytes slice is not a valid event representation")
	}

	prefix := messageBytes[:pipePos+1]
	jsonBytes := messageBytes[pipePos+1:]

	var event any
	var err error
	switch {
	case bytes.Equal(prefix, newMessagePrefix):
		event, err = unmarshal[NewMessage](jsonBytes)
	case bytes.Equal(prefix, systemMessagePrefix):
		event, err = unmarshal[SystemMessage](jsonBytes)
	default:
		return nil, fmt.Errorf("unexpected event prefix '%v'", prefix)
	}

	return event, err
}

func unmarshal[T any](bytes []byte) (*T, error) {
	result := new(T)
	err := json.Unmarshal(bytes, result)
	return result, err
}
