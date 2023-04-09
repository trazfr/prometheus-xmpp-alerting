package main

type Format int

const (
	Format_Text Format = iota
	Format_HTML
)

func (f Format) String() string {
	switch f {
	case Format_Text:
		return "text"
	case Format_HTML:
		return "html"
	default:
		return "unknown"
	}
}

// SendCloser is the interface to send messages
type SendCloser interface {
	Sender
	Close() error
}

// Sender is the interface to send messages
type Sender interface {
	Send(message string, format Format) error
}
