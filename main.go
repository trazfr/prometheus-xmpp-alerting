package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func getAddr(addr string) string {
	if addr == "" {
		return ":http"
	}
	return addr
}

func getNetwork(addr string) string {
	if strings.HasPrefix(addr, "/") {
		return "unix"
	}
	return "tcp"
}

func listenAndServe(s *http.Server) error {
	network := getNetwork(s.Addr)
	addr := getAddr(s.Addr)
	slog.Info("Create HTTP server", "addr", addr, "network", network)
	ln, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage", os.Args[0], "<config_file>")
		os.Exit(1)
	}

	config, err := NewConfig(os.Args[1])
	if err != nil {
		slog.Error("Cannot read the configuration file", "error", err)
		os.Exit(1)
	}

	if config.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	mux := http.NewServeMux()
	server := http.Server{
		Addr:    config.Listen,
		Handler: mux,
	}

	xmpp, err := NewXMPP(config, &server)
	if err != nil {
		slog.Error("Cannot start the XMPP client", "error", err)
		os.Exit(1)
	}
	defer xmpp.Close()

	mux.Handle("/alert", NewAlertHandler(config, xmpp))
	mux.Handle("/send", NewSendHandler(xmpp))
	mux.Handle("/metrics", promhttp.Handler())
	slog.Info("Server closed", "error", listenAndServe(&server))
}
