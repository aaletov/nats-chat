package types

import "time"

type ChatMessage struct {
	Time time.Time `json:"time"`
	Text string    `json:"text"`
}

type OnlineMessage struct {
	IsOnline bool `json:"isOnline"`
}
