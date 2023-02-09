package interfaces

type EventPreProcessor interface {
	// Pre-process incoming event.
	PreProcess(event any, producer Client) error
}
