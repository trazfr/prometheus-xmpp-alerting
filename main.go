package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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
	slog.Info("Server closed", "error", server.ListenAndServe())
}
