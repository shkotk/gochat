package interfaces

type Client interface {
	// Gets client identifier.
	ID() string

	// Gets incoming events channel.
	In() <-chan any

	// Gets outgoing events channel.
	Out() chan<- any

	// Gets channel to be signalled when client is done.
	Done() <-chan struct{}
}
