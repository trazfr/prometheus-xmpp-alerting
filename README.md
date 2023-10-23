# prometheus-xmpp-alerting

Basic XMPP Alertmanager Webhook Receiver for Prometheus

## Purpose

This repository has been made to receive Prometheus alerts on my Phone without relying on a third party provider.  
To do so I have installed on my Raspberry PI:

 - [Prometheus](https://prometheus.io/)
 - [Alertmanager](https://prometheus.io/docs/alerting/alertmanager/)
 - [Prosody](https://prosody.im/), an XMPP server

On my phone, I have just installed an XMPP client.

## Having a working Golang environment:

```bash
go install github.com/trazfr/prometheus-xmpp-alerting@latest
```

## Use

This program is configured through a JSON file.

To run, just `prometheus-xmpp-alerting config.json`

This example of configuration file shows:

 - the webhook listening on `127.0.0.1:9091`
 - when the instance is starting, it sends to everyone `Prometheus Monitoring Started`
 - it sends a different message depending on a `severity` label
 - it sends a message when an alert is resolved
 - the templates are in plain text. The possible values are `text` or `html` using [XEP-0071](https://xmpp.org/extensions/xep-0071.html) which is deprecated. If omitted, it defaults to `text`
 - the program uses the XMPP user `monitoring@example.com` with a password
 - when it is working, it has the status `Monitoring Prometheus...`
 - it doesn't use a TLS socket due to the `no_tls` flag. Actually it will use STARTTLS due to the server configuration
 - it doesn't check the TLS certificates thanks to `tls_insecure` (for some reason, it doesn't work on my Prosody install, but as I'm connecting to localhost, it doesn't matter)
 - each time it receives an alert, it sends a notification to
   - 2 XMPP accounts `on-duty-1@example.com` and `on-duty-2@example.com`
   - 1 MUC `monitoring-room-id@conference.example.com` using the nick `monitoring-bot`

```json
{
    "listen": "127.0.0.1:9091",
    "startup_message": "Prometheus Monitoring Started",
    "firing": "{{ if eq .Labels.severity \"error\" }}🔥{{ else if eq .Labels.severity \"warning\" }}💣{{ else }}💡{{ end }} Firing {{ .Labels.alertname }}\n{{ .Annotations.description }} since {{ .StartsAt }}\n{{ .GeneratorURL }}",
    "resolved": "{{ .Labels.alertname }} resolved",
    "format": "text",
    "xmpp": {
        "user": "monitoring@example.com",
        "password": "MyXmppPassword",
        "status": "Monitoring Prometheus...",
        "no_tls": true,
        "tls_insecure": true,
        "send_notif": [
            "on-duty-1@example.com",
            "on-duty-2@example.com"
        ],
        "send_muc": [
            {
                "room": "monitoring-room-id@conference.example.com",
                "nick": "monitoring-bot"
            }
        ]
    }
}
```

## Exotic DNS configuration

Usually, the admin creates DNS records to resolve the XMPP server.  
In some circumstances such records are not created.

The field `.xmpp.override_server` must be set to point to the right server:

```json
{
    "xmpp": {
        "override_server": "192.168.0.42:4212",
        // ...
    }
    // ...
}
```

## Features

This program uses HTTP with 3 different paths:

 - `/alert` is used by Prometheus' Alertmanager to send alerts
 - `/send` is mainly used for debugging or if one just want to send simple message from another program. To send a message:
   - `curl -H 'Content-Type: text/plain' -X POST <my_ip:port>/send -d 'my message'`
   - `curl -H 'Content-Type: text/html' -X POST <my_ip:port>/send -d '<p style="color:green;font-weight:bold;">Green text</p>'` if the client supports the deprecated XEP-0071
 - `/metrics` to be scrapped by Prometheus. It exposes some basic metrics
