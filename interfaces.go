package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Format
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

func (f *Format) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch strings.ToLower(s) {
	case "", "text":
		*f = Format_Text
	case "html":
		*f = Format_HTML
	default:
		return fmt.Errorf("unknown format: %s", s)
	}
	return nil
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
