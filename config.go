package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"text/template"
)

// Config is the internal configuration type
type Config struct {
	Listen         string
	Debug          bool
	StartupMessage string
	Firing         *template.Template
	Resolved       *template.Template
	XMPP           ConfigXMPP
}

// ConfigXMPP is the configuration for XMPP connection
type ConfigXMPP struct {
	User        string
	Password    string
	SendNotif   []string
	Status      string
	NoTLS       bool
	TLSInsecure bool
}

type internalConfig struct {
	Debug          bool               `json:"debug"`
	Listen         string             `json:"listen"`
	StartupMessage string             `json:"startup_message"`
	Firing         string             `json:"firing"`
	Resolved       string             `json:"resolved"`
	XMPP           internalConfigXMPP `json:"xmpp"`
}

type internalConfigXMPP struct {
	User        string   `json:"user"`
	Password    string   `json:"password"`
	SendNotif   []string `json:"send_notif"`
	Status      string   `json:"status"`
	NoTLS       bool     `json:"no_tls"`
	TLSInsecure bool     `json:"tls_insecure"`
}

// NewConfig reads the JSON file filename and generates a configuration
func NewConfig(filename string) *Config {
	fd, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer fd.Close()

	internalConfig := &internalConfig{
		Listen: ":9091",
		XMPP: internalConfigXMPP{
			Status: "Monitoring",
		},
	}
	if err := json.NewDecoder(fd).Decode(internalConfig); err != nil {
		log.Fatalln(err)
	}

	return internalConfig.parse()
}

func (i *internalConfig) parse() *Config {
	return &Config{
		Debug:          i.Debug,
		Listen:         i.Listen,
		StartupMessage: i.StartupMessage,
		Firing:         parseTemplate(i.Firing),
		Resolved:       parseTemplate(i.Resolved),
		XMPP:           i.XMPP.parse(),
	}
}

func (i *internalConfigXMPP) parse() ConfigXMPP {
	result := ConfigXMPP{
		User:        i.User,
		Password:    i.Password,
		SendNotif:   i.SendNotif,
		Status:      i.Status,
		NoTLS:       i.NoTLS,
		TLSInsecure: i.TLSInsecure,
	}
	sort.Strings(result.SendNotif)
	return result
}

func parseTemplate(tmpl string) *template.Template {
	if tmpl == "" {
		return nil
	}

	return template.Must(template.New("").Parse(tmpl))
}
