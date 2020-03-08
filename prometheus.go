package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	promNamespace = "xmpp"
)

var (
	promAlertTriggeredMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "alert_trigger_total",
		Help:      "Number of successful calls to the /alert webhook.",
	})
	promAlertsProcessedMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "alert_processed_total",
		Help:      "Number of alerts processed in the /alert webhook.",
	})
	promSendTriggeredMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "send_trigger_total",
		Help:      "Number of successful calls to the /send webhook.",
	})
	promMessagesSentMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "messages_sent_total",
		Help:      "Number of messages sent.",
	}, []string{"recipient"})
	promMessagesReceivedMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "messages_received_total",
		Help:      "Number of messages received.",
	}, []string{"recipient"})
)

func init() {
	prometheus.MustRegister(promAlertTriggeredMetric)
	prometheus.MustRegister(promAlertsProcessedMetric)
	prometheus.MustRegister(promSendTriggeredMetric)
	prometheus.MustRegister(promMessagesSentMetric)
	prometheus.MustRegister(promMessagesReceivedMetric)
}
