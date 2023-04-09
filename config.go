package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"
)

// Config is the internal configuration type
type Config struct {
	Listen         string
	Debug          bool
	StartupMessage string
	Firing         *template.Template
	Resolved       *template.Template
	Format         Format
	XMPP           ConfigXMPP
}

// ConfigXMPP is the configuration for XMPP connection
type ConfigXMPP struct {
	OverrideServer string
	User           string
	Password       string
	SendNotif      []string
	Status         string
	NoTLS          bool
	TLSInsecure    bool
}

type internalConfig struct {
	Debug          bool               `json:"debug"`
	Listen         string             `json:"listen"`
	StartupMessage string             `json:"startup_message"`
	Firing         string             `json:"firing"`
	Resolved       string             `json:"resolved"`
	Format         string             `json:"format"`
	XMPP           internalConfigXMPP `json:"xmpp"`
}

type internalConfigXMPP struct {
	OverrideServer string   `json:"override_server"`
	User           string   `json:"user"`
	Password       string   `json:"password"`
	SendNotif      []string `json:"send_notif"`
	Status         string   `json:"status"`
	NoTLS          bool     `json:"no_tls"`
	TLSInsecure    bool     `json:"tls_insecure"`
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
		Format:         parseFormat(i.Format),
		XMPP:           i.XMPP.parse(),
	}
}

func (i *internalConfigXMPP) parse() ConfigXMPP {
	result := ConfigXMPP{
		OverrideServer: i.OverrideServer,
		User:           i.User,
		Password:       i.Password,
		SendNotif:      i.SendNotif,
		Status:         i.Status,
		NoTLS:          i.NoTLS,
		TLSInsecure:    i.TLSInsecure,
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

func parseFormat(format string) Format {
	switch strings.ToLower(format) {
	case "", "text":
		return Format_Text
	case "html":
		return Format_HTML
	}
	panic(fmt.Sprintf("unknown format: %s", format))
}
