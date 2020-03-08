package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	libxmpp "github.com/mattn/go-xmpp"
)

const (
	xmppStatusChat  = "chat"
	xmppHelpMessage = `Help:
 - help
 - quit`
)

type xmpp struct {
	client    *libxmpp.Client
	debugMode bool
	channel   chan xmppMessage
	status    string
	sendNotif []string
}

type xmppMessage struct {
	message string
}

// NewXMPP create an XMPP connection. Use Close() to end it
func NewXMPP(config *Config) SendCloser {
	log.Printf("Connect to XMPP account %s\n", config.XMPP.User)
	options := libxmpp.Options{
		User:          config.XMPP.User,
		Password:      config.XMPP.Password,
		Debug:         config.Debug,
		NoTLS:         config.XMPP.NoTLS,
		Status:        xmppStatusChat,
		StatusMessage: config.XMPP.Status,
	}
	if config.XMPP.TLSInsecure {
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	client, err := options.NewClient()
	if err != nil {
		log.Fatalf("Could not connect to XMPP server: %s", err)
	}

	result := &xmpp{
		client:    client,
		debugMode: config.Debug,
		channel:   make(chan xmppMessage),
		status:    config.XMPP.Status,
		sendNotif: config.XMPP.SendNotif,
	}
	go result.runSender()
	go result.runReceiver()
	if config.StartupMessage != "" {
		result.Send(config.StartupMessage)
	}
	return result
}

func (x *xmpp) Send(message string) error {
	if message != "" {
		x.channel <- xmppMessage{
			message: message,
		}
	}
	return nil
}

func (x *xmpp) Close() error {
	close(x.channel)
	x.client.SendPresence(libxmpp.Presence{
		Show:   "unavailable",
		Status: "No monitoring",
	})
	return x.client.Close()
}

func (x *xmpp) runSender() {
	for payload := range x.channel {
		for _, sendNotif := range x.sendNotif {
			if err := x.sendTo(sendNotif, payload.message); err != nil {
				log.Printf("ERROR %s\n", err)
			}
		}
	}
}

func (x *xmpp) runReceiver() {
	for {
		stanza, err := x.client.Recv()
		if err != nil {
			if err == io.EOF {
				x.Close()
				return
			}
			log.Fatal(err)
		}
		x.debug("Stanza: %v\n", stanza)
		switch v := stanza.(type) {
		case libxmpp.Chat:
			x.handleChat(&v)
		case libxmpp.Presence:
			x.handlePresence(&v)
		}
	}
}

func (x *xmpp) isKnown(person string) bool {
	idx := sort.SearchStrings(x.sendNotif, person)
	return idx < len(x.sendNotif) && x.sendNotif[idx] == person
}

func (x *xmpp) handleChat(chat *libxmpp.Chat) {
	if chat.Text != "" {
		remoteUser := strings.Split(chat.Remote, "/")
		if len(remoteUser) == 2 && x.isKnown(remoteUser[0]) {
			x.debug("CHAT type=%s, remote=%s, text=%s\n", chat.Type, chat.Remote, chat.Text)
			x.handleCommand(remoteUser[0], chat.Text)
		} else {
			x.debug("Unknown user: %v\n", chat)
		}
	}
}

func (x *xmpp) handlePresence(presence *libxmpp.Presence) {
	switch presence.Type {
	case "":
	case "unavailable":
		// something puts us as unavailable
		if presence.From == x.client.JID() {
			x.client.SendOrg(fmt.Sprintf("<presence xml:lang='en'><show>%s</show><status>%s</status></presence>", xmppStatusChat, x.status))
		}
	case "subscribe":
		if x.isKnown(presence.From) {
			x.client.ApproveSubscription(presence.From)
			x.debug("Approved subscription to %s\n", presence.From)
		} else {
			x.client.RevokeSubscription(presence.From)
			x.debug("Revoked subscription to %s\n", presence.From)
		}
	default:
		x.debug("Unhandled presence: %v\n", presence)
	}
}

func (x *xmpp) handleCommand(from, command string) {
	promMessagesReceivedMetric.WithLabelValues(from).Inc()
	switch strings.ToLower(command) {
	case "quit":
		x.Close()
	case "help":
		x.Send(xmppHelpMessage)
	default:
		x.sendTo(from, fmt.Sprintf("Unknown command: %s", command))
	}
}

func (x *xmpp) sendTo(to, message string) error {
	promMessagesSentMetric.WithLabelValues(to).Inc()
	_, err := x.client.Send(libxmpp.Chat{
		Remote: to,
		Type:   "chat",
		Text:   message,
	})
	return err
}

func (x *xmpp) debug(fmt string, v ...interface{}) {
	if x.debugMode {
		log.Printf(fmt, v...)
	}
}
