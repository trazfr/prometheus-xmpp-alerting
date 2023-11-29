package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"text/template"
	"time"
)

// Config is the internal configuration type
type Config struct {
	Debug          bool            `json:"debug"`
	Listen         string          `json:"listen"`
	StartupMessage string          `json:"startup_message"`
	Firing         *ConfigTemplate `json:"firing"`
	Resolved       *ConfigTemplate `json:"resolved"`
	Format         Format          `json:"format"`
	XMPP           ConfigXMPP      `json:"xmpp"`
	TimeZone       *ConfigTimeZone `json:"time_zone"`
}

// ConfigXMPP is the configuration for XMPP connection
type ConfigXMPP struct {
	OverrideServer string       `json:"override_server"`
	User           string       `json:"user"`
	Password       string       `json:"password"`
	SendNotif      []string     `json:"send_notif"`
	SendMUC        []*ConfigMUC `json:"send_muc"`
	Status         string       `json:"status"`
	NoTLS          bool         `json:"no_tls"`
	TLSInsecure    bool         `json:"tls_insecure"`
}

// ConfigMUC is the list of MUC to join (xep-0045)
type ConfigMUC struct {
	Room     string  `json:"room"`
	Nick     string  `json:"nick"`
	Password *string `json:"password"`
}

// NewConfig reads the JSON file filename and generates a configuration
func NewConfig(filename string) *Config {
	fd, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer fd.Close()

	config := &Config{
		Listen: ":9091",
		XMPP: ConfigXMPP{
			Status: "Monitoring",
		},
	}
	if err := json.NewDecoder(fd).Decode(config); err != nil {
		log.Fatalln(err)
	}

	// default nick
	for _, configMUC := range config.XMPP.SendMUC {
		if configMUC.Nick == "" {
			configMUC.Nick = strings.Split(config.XMPP.User, "@")[0]
		}
	}

	return config
}

// ConfigTemplate

type ConfigTemplate template.Template

func (c *ConfigTemplate) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	templ, err := template.New("").Parse(s)
	if err == nil {
		*c = ConfigTemplate(*templ)
	}
	return err
}

func (c *ConfigTemplate) ToTemplate() *template.Template {
	return (*template.Template)(c)
}

// ConfigTimeZone

type ConfigTimeZone time.Location

func (c *ConfigTimeZone) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	loc, err := time.LoadLocation(s)
	if err == nil {
		*c = ConfigTimeZone(*loc)
	}
	return err
}

func (c *ConfigTimeZone) ToLocation() *time.Location {
	return (*time.Location)(c)
}
