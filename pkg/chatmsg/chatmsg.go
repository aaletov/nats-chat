package chatmsg

import "time"

type ChatMessage struct {
	Time time.Time `json:"time"`
	Text string    `json:"text"`
}
