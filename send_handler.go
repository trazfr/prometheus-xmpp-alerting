package main

import (
	"io/ioutil"
	"log"
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
	if data, err := ioutil.ReadAll(r.Body); err == nil {
		promSendTriggeredMetric.Inc()
		s.sender.Send(string(data), s.getFormat(r.Header.Get("content-type")))
	} else {
		log.Printf("sendHandler: could not read the body: %s\n", err)
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
	log.Printf("Unknown content-type: %s. Using the default: text/plain", contentType)
	return Format_Text
}
