package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage", os.Args[0], "<config_file>")
		os.Exit(1)
	}

	config := NewConfig(os.Args[1])
	xmpp := NewXMPP(config)
	defer xmpp.Close()

	http.Handle("/alert", NewAlertHandler(config, xmpp))
	http.Handle("/send", NewSendHandler(xmpp))
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println(http.ListenAndServe(config.Listen, nil))
}
