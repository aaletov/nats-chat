package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aaletov/nats-chat/pkg/profiles"
	"github.com/aaletov/nats-chat/pkg/types"
	"github.com/nats-io/nats.go"
)

type Session struct {
	nc      *nats.Conn
	profile profiles.SenderProfile
	pingSub *nats.Subscription
}

func NewSession(nc *nats.Conn, profile profiles.SenderProfile) Session {
	return Session{
		nc:      nc,
		profile: profile,
	}
}

// There should be a way to handle s == nil
func (s *Session) Open() error {
	senderPing := fmt.Sprintf("ping.%s", s.profile.GetAddress())
	senderOnline := fmt.Sprintf("online.%s", s.profile.GetAddress())

	sub, _ := s.nc.Subscribe(senderPing, func(msg *nats.Msg) {
		omsg, _ := json.Marshal(types.OnlineMessage{IsOnline: true})
		s.nc.Publish(senderOnline, omsg)
		msg.Ack()
	})
	s.pingSub = sub
	log.Printf("Subscribed at sender ping: %s\n", senderPing)
	return nil
}

func (s *Session) Close() error {
	return s.pingSub.Unsubscribe()
}

func (s *Session) Dial(recepient string) (*ChatConnection, error) {
	recepientPing := fmt.Sprintf("ping.%s", recepient)
	recepientOnline := fmt.Sprintf("online.%s", recepient)
	recepientChat := fmt.Sprintf("chat.%s", recepient)
	senderChat := fmt.Sprintf("chat.%s", s.profile.GetAddress())

	online := make(chan bool)
	incomingChan := make(chan types.ChatMessage)
	onlineSub, _ := s.nc.Subscribe(recepientOnline, func(msg *nats.Msg) {
		var omsg types.OnlineMessage
		if err := json.Unmarshal(msg.Data, &omsg); err != nil {
			msg.Nak()
			return
		}
		online <- omsg.IsOnline
	})
	log.Printf("Subscribed at recepient online: %s\n", recepientOnline)

	chatSub, _ := s.nc.Subscribe(senderChat, func(msg *nats.Msg) {
		var cmsg types.ChatMessage
		if err := json.Unmarshal(msg.Data, &cmsg); err != nil {
			msg.Nak()
			return
		}
		incomingChan <- cmsg
		msg.Ack()
	})
	log.Printf("Subscribed at sender chat %s\n", senderChat)

	ticker := time.NewTicker(33 * time.Millisecond)
	err := func() error {
		for {
			select {
			case <-ticker.C:
				s.nc.Publish(recepientPing, []byte{})
				log.Printf("Pinged %s\n", recepient)
			case isOnline := <-online:
				ticker.Stop()
				if isOnline {
					return nil
				} else {
					return errors.New("Recepient went offline")
				}
			}
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("unable to dial %s: %s", recepient, err)
	}

	outcomingChan := make(chan types.ChatMessage)
	go func() {
		for cmsg := range outcomingChan {
			var (
				err  error
				data []byte
			)
			if data, err = json.Marshal(cmsg); err != nil {
				log.Printf("unable to marshal message: %s", cmsg)
			}
			s.nc.Publish(recepientChat, data)
			log.Printf("Sent message %s\n", cmsg)
		}
	}()

	return &ChatConnection{
		SenderAddress:    s.profile.GetAddress(),
		RecepientAddress: recepient,
		OnlineChan:       online,
		IncomingChan:     incomingChan,
		OutcomingChan:    outcomingChan,
		onlineSub:        onlineSub,
		chatSub:          chatSub,
		nc:               s.nc,
	}, nil
}

type ChatConnection struct {
	SenderAddress    string
	RecepientAddress string
	OnlineChan       chan bool
	IncomingChan     chan types.ChatMessage
	OutcomingChan    chan types.ChatMessage
	onlineSub        *nats.Subscription
	chatSub          *nats.Subscription
	nc               *nats.Conn
}

func (c *ChatConnection) Close() (err error) {
	senderOnline := fmt.Sprintf("online.%s", c.SenderAddress)
	offlineMsg := types.OnlineMessage{IsOnline: false}
	data, _ := json.Marshal(offlineMsg)
	c.nc.Publish(senderOnline, data)

	if err = c.onlineSub.Unsubscribe(); err != nil {
		return err
	}
	if err = c.chatSub.Unsubscribe(); err != nil {
		return err
	}
	close(c.IncomingChan)
	return nil
}
