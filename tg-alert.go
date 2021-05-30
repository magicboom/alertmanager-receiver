package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"flag"
	"io/ioutil"

	"github.com/go-yaml/yaml"
	botapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	timeDateFormat = "2006-01-02 15:04:05"
)
var (
	argConfigFile = flag.String("c", "", "Config file (yaml format)")
	cfg Config
)

type Config struct {
	BotToken string `yaml:"bot_token"`
	ChatID   int64  `yaml:"chat_id"`
	ListenTo string `yaml:"listen"`
	WebPath  string `yaml:"web_path"`
}

type alertmanagerAlert struct {
	Receiver string `json:"receiver"`
	Status   string `json:"status"`
	Alerts   []struct {
		Status string `json:"status"`
		Labels struct {
			Name      string `json:"name"`
			Instance  string `json:"instance"`
			Alertname string `json:"alertname"`
			Service   string `json:"service"`
			Severity  string `json:"severity"`
		} `json:"labels"`
		Annotations struct {
			Info        string `json:"info"`
			Description string `json:"description"`
			Summary     string `json:"summary"`
		} `json:"annotations"`
		StartsAt     time.Time `json:"startsAt"`
		EndsAt       time.Time `json:"endsAt"`
		GeneratorURL string    `json:"generatorURL"`
		Fingerprint  string    `json:"fingerprint"`
	} `json:"alerts"`
	GroupLabels struct {
		Alertname string `json:"alertname"`
	} `json:"groupLabels"`
	CommonLabels struct {
		Alertname string `json:"alertname"`
		Service   string `json:"service"`
		Severity  string `json:"severity"`
	} `json:"commonLabels"`
	CommonAnnotations struct {
		Summary string `json:"summary"`
	} `json:"commonAnnotations"`
	ExternalURL string `json:"externalURL"`
	Version     string `json:"version"`
	GroupKey    string `json:"groupKey"`
}

// ToTelegram function responsible to send msg to telegram
func ToTelegram(w http.ResponseWriter, r *http.Request) {

	var alerts alertmanagerAlert

	bot, err := botapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic(err)
	}

	_ = json.NewDecoder(r.Body).Decode(&alerts)

	for _, alert := range alerts.Alerts {
		telegramMsg := "Status: " + alert.Status + "\n"
		if alert.Labels.Name != "" {
			telegramMsg += "Instance: " + alert.Labels.Instance + "(" + alert.Labels.Name + ")\n"
		}
		if alert.Annotations.Info != "" {
			telegramMsg += "Info: " + alert.Annotations.Info + "\n"
		}
		if alert.Annotations.Summary != "" {
			telegramMsg += "Summary: " + alert.Annotations.Summary + "\n"
		}
		if alert.Annotations.Description != "" {
			telegramMsg += "Description: " + alert.Annotations.Description + "\n"
		}
		if alert.Status == "resolved" {
			telegramMsg += "Resolved: " + alert.EndsAt.Format(timeDateFormat)
		} else if alert.Status == "firing" {
			telegramMsg += "Started: " + alert.StartsAt.Format(timeDateFormat)
		}

		msg := botapi.NewMessage(-cfg.ChatID, telegramMsg)
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("ERR: Unable to send message: %v, error: %v", msg, err)
		}
	}

	log.Println(alerts)
	json.NewEncoder(w).Encode(alerts)

}

func main() {
    flag.Parse()
    if *argConfigFile == "" {
	log.Fatal("arg '-c <config-file>' required")
    }

    dat, err := ioutil.ReadFile(*argConfigFile)
    if err != nil {
	panic(err)
    }
    err = yaml.Unmarshal([]byte(dat), &cfg)

    toTgHandler := func(w http.ResponseWriter, req *http.Request) {
        ToTelegram(w, req)
    }
    log.Printf("Waiting for prometheus alerts on '%s%s'", cfg.ListenTo, cfg.WebPath)
    http.HandleFunc(cfg.WebPath, toTgHandler)
    s := &http.Server{
        Addr:           cfg.ListenTo,
        ReadTimeout:    10 * time.Second,
        WriteTimeout:   10 * time.Second,
        MaxHeaderBytes: 1 << 20,
    }
    log.Fatal(s.ListenAndServe())

}
