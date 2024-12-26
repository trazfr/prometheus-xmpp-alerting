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
	xmpp, err := NewXMPP(config)
	if err != nil {
		slog.Error("Cannot start the XMPP client", "error", err)
		os.Exit(1)
	}
	defer xmpp.Close()

	http.Handle("/alert", NewAlertHandler(config, xmpp))
	http.Handle("/send", NewSendHandler(xmpp))
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println(http.ListenAndServe(config.Listen, nil))
}
