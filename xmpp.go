package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	libxmpp "github.com/xmppo/go-xmpp"
)

const (
	xmppStatusChat  = "chat"
	xmppHelpMessage = `Help:
 - help
 - metrics
 - ping`
)

type xmpp struct {
	client        *libxmpp.Client
	channel       chan xmppMessage
	status        string
	sendNotif     []string
	sendMUC       []string
	closeCallback io.Closer
	closed        atomic.Bool
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
func NewXMPP(config *Config, closeCallback io.Closer) (SendCloser, error) {
	if config.XMPP.OverrideServer != "" {
		slog.Info("Connect to the XMPP account", "user", config.XMPP.User, "method", "override", "server", config.XMPP.OverrideServer)
	} else {
		slog.Info("Connect to the XMPP account", "user", config.XMPP.User, "method", "DNS SRV")
	}
	options := libxmpp.Options{
		Host:          config.XMPP.OverrideServer,
		User:          config.XMPP.User,
		Password:      config.XMPP.Password,
		Debug:         config.Debug,
		NoTLS:         !config.XMPP.TLS,
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
		return nil, fmt.Errorf("could not connect to XMPP server: %w", err)
	}

	result := &xmpp{
		client:        client,
		channel:       make(chan xmppMessage),
		status:        config.XMPP.Status,
		sendNotif:     config.XMPP.SendNotif,
		closeCallback: closeCallback,
		closed:        atomic.Bool{},
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
			client.Close()
			return nil, fmt.Errorf("could not connect to MUC: %w", err)
		}
	}

	prometheus.MustRegister(result)

	go result.runSender()
	go result.runReceiver()
	if config.StartupMessage != "" {
		result.Send(config.StartupMessage, config.Format)
	}
	return result, nil
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
	if x.closed.Swap(true) {
		return nil
	}

	prometheus.Unregister(x)
	close(x.channel)
	x.client.SendPresence(libxmpp.Presence{
		Show:   "unavailable",
		Status: "No monitoring",
	})
	if err := x.client.Close(); err != nil {
		slog.Error("Could not close the XMPP connection", "error", err)
	}
	return x.closeCallback.Close()
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
	errorNum := 0
	for {
		stanza, err := x.client.Recv()
		if err != nil {
			if err == io.EOF {
				slog.Error("XMPP connection closed", "error", err)
				x.Close()
				return
			}
			errorNum++
			slog.Error("Could not receive the XMPP message", "error", err)
			if errorNum > 5 {
				slog.Error("Too many errors. Closing the XMPP connection")
				os.Exit(1)
			}
			time.Sleep(1 * time.Second)
			continue
		}
		errorNum = 0
		slog.Debug("Stanza: %v", stanza)
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
			slog.Debug("CHAT", "type", chat.Type, "remote", chat.Remote, "text", chat.Text)
			x.handleCommand(remoteUser[0], chat.Text)
		} else {
			slog.Debug("Unknown user", "user", chat)
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
			slog.Debug("Approved subscription", "user", presence.From)
		} else {
			x.client.RevokeSubscription(presence.From)
			slog.Debug("Revoked subscription", "user", presence.From)
		}
	case "error":
		slog.Info("Error", "user", presence.From)
	default:
		slog.Debug("Unhandled presence", "type", presence.Type, "from", presence.From)
	}
}

func (x *xmpp) handleCommand(from, command string) {
	promMessagesReceivedMetric.WithLabelValues(from).Inc()
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "ping":
		x.sendTo(from, "pong")
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
		slog.Error("Cannot send the XMPP message", "error", err)
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
