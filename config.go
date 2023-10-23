package main

import (
	"encoding/json"
	"log"
	"os"
	"text/template"
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
}

// ConfigXMPP is the configuration for XMPP connection
type ConfigXMPP struct {
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

	config := &Config{
		Listen: ":9091",
		XMPP: ConfigXMPP{
			Status: "Monitoring",
		},
	}
	if err := json.NewDecoder(fd).Decode(config); err != nil {
		log.Fatalln(err)
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
