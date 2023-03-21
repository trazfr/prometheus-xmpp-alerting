package main

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

const (
	promNamespace = "xmpp"
)

var (
	promInfo = prometheus.NewDesc(
		promNamespace+"_info",
		"constant metric with value=1. Various information about the XMPP connection.",
		[]string{"encrypted", "jid"}, nil)
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

func getMetrics() ([]string, error) {
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return nil, err
	}
	var result []string = nil
	var labelsSlice []string = nil
	for _, metricFamily := range metricFamilies {
		for _, metric := range metricFamily.GetMetric() {
			if labelsSlice != nil {
				labelsSlice = labelsSlice[:0]
			}
			for _, label := range metric.GetLabel() {
				labelsSlice = append(labelsSlice, fmt.Sprintf(`%s="%s"`, label.GetName(), label.GetValue()))
			}
			metricType, value := getMetricsValue(metric)
			result = append(result, fmt.Sprintf("%s %s{%s}: %f", metricType, metricFamily.GetName(), strings.Join(labelsSlice, ","), value))
		}
	}
	return result, nil
}

func getMetricsValue(metric *io_prometheus_client.Metric) (string, float64) {
	if counter := metric.GetCounter(); counter != nil {
		return "counter", counter.GetValue()
	}
	if gauge := metric.GetGauge(); gauge != nil {
		return "gauge", gauge.GetValue()
	}
	if summary := metric.GetSummary(); summary != nil {
		return "summary", summary.GetSampleSum() / float64(summary.GetSampleCount())
	}
	if untyped := metric.GetUntyped(); untyped != nil {
		return "untyped", untyped.GetValue()
	}
	if histogram := metric.GetHistogram(); histogram != nil {
		return "histogram", histogram.GetSampleSum() / float64(histogram.GetSampleCount())
	}
	return "unknown", 0
}
