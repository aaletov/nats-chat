package types

import "time"

type ChatMessage struct {
	Time time.Time `json:"time"`
	Text string    `json:"text"`
}

type OnlineMessage struct {
	AuthorAddress string `json:"authorAddress"`
	IsOnline      bool   `json:"isOnline"`
}

type PingMessage struct {
	AuthorAddress string `json:"authorAddress"`
}
