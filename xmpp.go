package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	libxmpp "github.com/xmppo/go-xmpp"
)

const (
	xmppStatusChat  = "chat"
	xmppHelpMessage = `Help:
 - help
 - metrics
 - quit`
)

type xmpp struct {
	client    *libxmpp.Client
	debugMode bool
	channel   chan xmppMessage
	status    string
	sendNotif []string
	sendMUC   []string
}

type xmppMessage struct {
	to      *string
	message string
	format  Format
}

type xmppChatType int

const (
	xmppChatType_Chat xmppChatType = iota
	xmppChatType_GroupChat
)

func (x xmppChatType) String() string {
	switch x {
	case xmppChatType_GroupChat:
		return "groupchat"
	case xmppChatType_Chat:
		return "chat"
	default:
		return "chat"
	}
}

func xmppChatTypeFrom(s string) (xmppChatType, error) {
	switch s {
	case xmppChatType_Chat.String():
		return xmppChatType_Chat, nil
	case xmppChatType_GroupChat.String():
		return xmppChatType_GroupChat, nil
	default:
		return xmppChatType_Chat, fmt.Errorf("unhandled chat type: %s", s)
	}
}

// NewXMPP create an XMPP connection. Use Close() to end it
func NewXMPP(config *Config) SendCloser {
	if config.XMPP.OverrideServer != "" {
		log.Println("Connect to the XMPP account", config.XMPP.User, "using the server", config.XMPP.OverrideServer)
	} else {
		log.Println("Connect to the XMPP account", config.XMPP.User, "using a server from the DNS records")
	}
	options := libxmpp.Options{
		Host:          config.XMPP.OverrideServer,
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

	for _, muc := range config.XMPP.SendMUC {
		var err error
		result.sendMUC = append(result.sendMUC, muc.Room)
		if muc.Password != nil {
			_, err = client.JoinProtectedMUC(muc.Room, muc.Nick, *muc.Password, libxmpp.NoHistory, 0, nil)
		} else {
			_, err = client.JoinMUC(muc.Room, muc.Nick, libxmpp.NoHistory, 0, nil)
		}
		if err != nil {
			log.Fatalf("Could not connect to MUC: %s", err)
		}
	}

	prometheus.MustRegister(result)

	go result.runSender()
	go result.runReceiver()
	if config.StartupMessage != "" {
		result.Send(config.StartupMessage, config.Format)
	}
	return result
}

func (x *xmpp) Send(message string, format Format) error {
	if message != "" {
		x.channel <- xmppMessage{
			message: message,
			format:  format,
		}
	}
	return nil
}

func (x *xmpp) Close() error {
	prometheus.Unregister(x)
	close(x.channel)
	x.client.SendPresence(libxmpp.Presence{
		Show:   "unavailable",
		Status: "No monitoring",
	})
	return x.client.Close()
}

func (x *xmpp) sendTo(to, message string) error {
	if message != "" {
		x.channel <- xmppMessage{
			to:      &to,
			message: message,
			format:  Format_Text,
		}
	}
	return nil
}

func (x *xmpp) runSender() {
	for payload := range x.channel {
		if payload.to != nil {
			x.sendToImmediate(xmppChatType_Chat, *payload.to, payload.message, payload.format)
		} else {
			for _, sendNotif := range x.sendNotif {
				x.sendToImmediate(xmppChatType_Chat, sendNotif, payload.message, payload.format)
			}
			for _, room := range x.sendMUC {
				x.sendToImmediate(xmppChatType_GroupChat, room, payload.message, payload.format)
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
			chatType, err := xmppChatTypeFrom(v.Type)
			if err == nil && chatType == xmppChatType_Chat {
				x.handleChat(&v)
			}
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
	case "error":
		fmt.Printf("Error from %s", presence.From)
	default:
		x.debug("Unhandled presence: %v\n", presence)
	}
}

func (x *xmpp) handleCommand(from, command string) {
	promMessagesReceivedMetric.WithLabelValues(from).Inc()
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "quit":
		x.Close()
	case "metrics":
		if metrics, err := getMetrics(); err == nil {
			for _, metric := range metrics {
				x.sendTo(from, metric)
			}
		} else {
			x.sendTo(from, fmt.Sprintf("Could not fetch the metrics: %s", err))
		}
	case "help":
		x.sendTo(from, xmppHelpMessage)
	default:
		x.sendTo(from, fmt.Sprintf("Unknown command: %s\n%s", command, xmppHelpMessage))
	}
}

func (x *xmpp) sendToImmediate(chatType xmppChatType, to, message string, format Format) {
	promMessagesSentMetric.WithLabelValues(to, chatType.String(), format.String()).Inc()
	_, err := x.sendChat(libxmpp.Chat{
		Remote: to,
		Type:   chatType.String(),
		Text:   message,
	}, format)
	if err != nil {
		log.Printf("ERROR %s\n", err)
	}
}

func (x *xmpp) debug(fmt string, v ...interface{}) {
	if x.debugMode {
		log.Printf(fmt, v...)
	}
}

func (x *xmpp) sendChat(chat libxmpp.Chat, format Format) (n int, err error) {
	switch format {
	case Format_Text:
		return x.client.Send(chat)
	case Format_HTML:
		return x.client.SendHtml(chat)
	default:
		return 0, fmt.Errorf("unknown format: %d", format)
	}
}

// prometheus Collector

func (x *xmpp) Describe(ch chan<- *prometheus.Desc) {
	ch <- promInfo
}

func (x *xmpp) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(promInfo, prometheus.GaugeValue, 1, strconv.FormatBool(x.client.IsEncrypted()), x.client.JID())
}
