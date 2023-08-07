package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	api "github.com/aaletov/nats-chat/api/generated"
	"github.com/aaletov/nats-chat/pkg/types"
	"github.com/hashicorp/go-multierror"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type Session struct {
	logger        *logrus.Entry
	nc            *nats.Conn
	senderAddress string
	pingSub       *nats.Subscription
}

func Online(logger *logrus.Logger, nc *nats.Conn, senderAddress string) (*Session, error) {
	ll := logger.WithFields(logrus.Fields{
		"method": "Online",
	})
	senderPing := fmt.Sprintf("ping.%s", senderAddress)
	var err error
	sub, err := nc.Subscribe(senderPing, func(msg *nats.Msg) {
		var (
			err        error
			pmsg       types.PingMessage
			omsg       types.OnlineMessage
			marshalled []byte
		)

		if err = json.Unmarshal(msg.Data, &pmsg); err != nil {
			ll.Printf("error unmarshalling ping message: %s\n", err)
		}
		recepientOnline := fmt.Sprintf("online.%s", pmsg.AuthorAddress)
		omsg = types.OnlineMessage{AuthorAddress: senderAddress, IsOnline: true}
		if marshalled, err = json.Marshal(omsg); err != nil {
			ll.Printf("error marshalling online message: %s\n", err)
		}
		nc.Publish(recepientOnline, marshalled)
		msg.Ack()
	})
	if err != nil {
		return nil, fmt.Errorf("error subscribing to ping: %s", err)
	}

	ll.Printf("Subscribed at sender ping: %s\n", senderPing)
	return &Session{
		logger:        logger.WithFields(logrus.Fields{"component": "Session"}),
		nc:            nc,
		senderAddress: senderAddress,
		pingSub:       sub,
	}, nil
}

func (s *Session) Close() (err error) {
	s.nc.Close()
	return s.pingSub.Unsubscribe()
}

func NewIncomingMsgHandler(logger *logrus.Logger, incomingChan chan *api.ChatMessage) nats.MsgHandler {
	return func(msg *nats.Msg) {
		cmsg := &api.ChatMessage{}
		if err := json.Unmarshal(msg.Data, cmsg); err != nil {
			logger.Fatalf("Error unmarshalling message: %s", err)
			msg.Nak()
			return
		}
		logger.Debugf("Got message from nats in handler: %s", cmsg)
		incomingChan <- cmsg
		msg.Ack()
		logger.Debugln("Acknowledged nats")
	}
}

// func ProccessOutcomingChan(nc *nats.Conn, recepient string, outcomingChan chan api.ChatMessage) {
// 	recepientChat := fmt.Sprintf("chat.%s", c.RecepientAddress)
// 	for cmsg := <- outcomingChan {
// 		nc.Publish(recepient, cmsg)
// 	}
// }

func (s *Session) Dial(recepient string) (*ChatConnection, error) {
	ll := s.logger.WithFields(logrus.Fields{
		"method": "Dial",
	})
	senderOnline := fmt.Sprintf("online.%s", s.senderAddress)
	senderChat := fmt.Sprintf("chat.%s", s.senderAddress)
	recepientPing := fmt.Sprintf("ping.%s", recepient)

	var err error
	online := make(chan bool)
	onlineSub, err := s.nc.Subscribe(senderOnline, func(msg *nats.Msg) {
		var omsg types.OnlineMessage
		if err := json.Unmarshal(msg.Data, &omsg); err != nil {
			msg.Nak()
			return
		}
		if omsg.AuthorAddress != recepient {
			msg.Nak()
			return
		}
		online <- omsg.IsOnline
		msg.Ack()
	})
	if err != nil {
		return nil, fmt.Errorf("error subscribing to sender online: %s", err)
	}
	ll.Debugf("Subscribed at sender online: %s\n", senderOnline)

	incomingChan := make(chan *api.ChatMessage)
	chatSub, _ := s.nc.Subscribe(senderChat, NewIncomingMsgHandler(ll.Logger, incomingChan))
	ll.Debugf("Subscribed at sender chat %s\n", senderChat)

	ticker := time.NewTicker(33 * time.Millisecond)
	err = func() error {
		pmsg := types.PingMessage{AuthorAddress: s.senderAddress}
		var (
			err  error
			data []byte
		)
		if data, err = json.Marshal(pmsg); err != nil {
			ll.Debugf("error marshal ping message: %s\n", err)
		}
		for {
			select {
			case <-ticker.C:
				s.nc.Publish(recepientPing, data)
				ll.Debugf("Pinged %s\n", recepient)
			case isOnline := <-online:
				ticker.Stop()
				if !isOnline {
					return errors.New("Recepient went offline")
				}
				ll.Debugf("Got online from %s", recepient)
				return nil
			}
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("unable to dial %s: %s", recepient, err)
	}

	return &ChatConnection{
		logger: s.logger.Logger.WithFields(logrus.Fields{
			"component": "ChatConnection",
		}),
		SenderAddress:    s.senderAddress,
		RecepientAddress: recepient,
		incomingChan:     incomingChan,
		onlineSub:        onlineSub,
		chatSub:          chatSub,
		nc:               s.nc,
	}, nil
}

type ChatConnection struct {
	logger           *logrus.Entry
	SenderAddress    string
	RecepientAddress string
	incomingChan     chan *api.ChatMessage
	onlineSub        *nats.Subscription
	chatSub          *nats.Subscription
	nc               *nats.Conn
}

func (c *ChatConnection) Send(srv api.Daemon_SendServer) error {
	ll := c.logger.WithFields(logrus.Fields{
		"method": "Send",
	})
	recepientChat := fmt.Sprintf("chat.%s", c.RecepientAddress)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		for cmsg := range c.incomingChan {
			ll.Debugf("Got message from nats: %s", cmsg)
			if err = srv.Send(cmsg); err != nil {
				ll.Fatalf("Unable to send message: %s\n", err)
			}
			ll.Debugf("Sent message to cli: %s", cmsg)
		}
		ll.Debugln("Exiting server send loop")
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var (
			err  error
			cmsg *api.ChatMessage
			data []byte
		)
		for {
			if cmsg, err = srv.Recv(); err != nil {
				ll.Fatalf("Unable to get message: %s", err)
			}
			ll.Debugf("Got message from cli: %s", cmsg)
			// Use generated marshallers
			if data, err = json.Marshal(cmsg); err != nil {
				ll.Fatalf("unable to marshal message: %s\n", err)
			}
			c.nc.Publish(recepientChat, data)
			ll.Debugf("Published message: %s", cmsg)
		}
		ll.Debugln("Exiting server recv loop")
	}()
	wg.Wait()
	return nil
}

func (c *ChatConnection) Close() (err error) {
	ll := c.logger.WithFields(logrus.Fields{
		"method": "Close",
	})
	ll.Printf("Closing ChatConnection %s\n", c.RecepientAddress)
	recepientOnline := fmt.Sprintf("online.%s", c.RecepientAddress)
	offlineMsg := types.OnlineMessage{IsOnline: false, AuthorAddress: c.SenderAddress}
	data, err := json.Marshal(offlineMsg)
	if err != nil {
		return err
	}
	close(c.incomingChan)
	err = multierror.Append(c.nc.Publish(recepientOnline, data))
	err = multierror.Append(err, c.onlineSub.Unsubscribe())
	err = multierror.Append(err, c.chatSub.Unsubscribe())
	return err
}
