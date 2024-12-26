package main

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unicode"
)

type sendHandler struct {
	sender Sender
}

// NewSendHandler create an HTTP handler which copies the body into the sender
func NewSendHandler(sender Sender) http.Handler {
	return &sendHandler{sender}
}

func (s *sendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if data, err := io.ReadAll(r.Body); err == nil {
		promSendTriggeredMetric.Inc()
		s.sender.Send(string(data), s.getFormat(r.Header.Get("content-type")))
	} else {
		slog.Error("sendHandler: could not read the body", "error", err)
	}
}

func (s *sendHandler) getFormat(contentType string) Format {
	contentType = strings.TrimFunc(strings.Split(contentType, ";")[0], unicode.IsSpace)
	switch contentType {
	case "text/plain":
		return Format_Text
	case "text/html", "application/xhtml+xml", "application/xml", "text/xml":
		return Format_HTML
	}
	slog.Error("Unknown content-type. Using the default: text/plain", "content-type", contentType)
	return Format_Text
}
