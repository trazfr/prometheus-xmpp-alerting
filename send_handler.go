package main

import (
	"io/ioutil"
	"log"
	"net/http"
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
		s.sender.Send(string(data))
	} else {
		log.Printf("sendHandler: could not read the body: %s\n", err)
	}
}
