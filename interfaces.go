package main

// SendCloser is the interface to send messages
type SendCloser interface {
	Sender
	Close() error
}

// Sender is the interface to send messages
type Sender interface {
	Send(message string) error
}
