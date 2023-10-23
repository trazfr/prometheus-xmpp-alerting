package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"text/template"

	promTemplate "github.com/prometheus/alertmanager/template"
)

type alertHandler struct {
	sender           Sender
	firingTemplate   *template.Template
	resolvedTemplate *template.Template
	format           Format
}

// NewAlertHandler create an HTTP handler to receive prometheus webhook alerts
func NewAlertHandler(config *Config, sender Sender) http.Handler {
	return &alertHandler{
		sender:           sender,
		firingTemplate:   config.Firing.ToTemplate(),
		resolvedTemplate: config.Resolved.ToTemplate(),
		format:           config.Format,
	}
}

func (a *alertHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// https://godoc.org/github.com/prometheus/alertmanager/template#Data
	promAlert := promTemplate.Data{}
	if err := json.NewDecoder(r.Body).Decode(&promAlert); err != nil {
		a.handleError(w, http.StatusBadRequest, err, "Cannot decode payload")
		return
	}
	promAlertTriggeredMetric.Inc()

	a.instantiateTemplate(a.firingTemplate, promAlert.Alerts.Firing())
	a.instantiateTemplate(a.resolvedTemplate, promAlert.Alerts.Resolved())
}

func (a *alertHandler) handleError(w http.ResponseWriter, statusCode int, err error, message string) {
	w.WriteHeader(statusCode)
	w.Write([]byte("Error: "))
	w.Write([]byte(err.Error()))
	if message != "" {
		w.Write([]byte("\n"))
		w.Write([]byte(message))
	}
}

func (a *alertHandler) instantiateTemplate(tmpl *template.Template, alerts []promTemplate.Alert) {
	if tmpl == nil {
		return
	}

	for alertIdx := range alerts {
		if message := a.generateString(tmpl, &alerts[alertIdx]); message != "" {
			promAlertsProcessedMetric.Inc()
			a.sender.Send(message, a.format)
		}
	}
}

func (a *alertHandler) generateString(tmpl *template.Template, alert *promTemplate.Alert) string {
	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, alert); err != nil {
		log.Printf("Could not instantiate template :%s\n", err)
		return ""
	}
	return buf.String()
}
