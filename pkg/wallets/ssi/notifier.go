package ssi

import (
	"encoding/json"
	"github.com/allinbits/cosmos-cash-agent/pkg/config"
	"github.com/allinbits/cosmos-cash-agent/pkg/model"
	"github.com/hyperledger/aries-framework-go/pkg/framework/context"
	"github.com/hyperledger/aries-framework-go/pkg/store/connection"
	log "github.com/sirupsen/logrus"
)

// LocalNotifier handles message events
type LocalNotifier struct {
	RuntimeMsgs      *config.MsgHub
	ConnectionLookup *connection.Lookup
}

// NewNotifier return notifier instance.
func NewNotifier(ctx *context.Provider, RuntimeMsgs *config.MsgHub) *LocalNotifier {
	connectionLookup, err := connection.NewLookup(ctx)
	if err != nil {
		log.Errorln(err)
	}
	return &LocalNotifier{
		RuntimeMsgs:      RuntimeMsgs,
		ConnectionLookup: connectionLookup,
	}
}

// Notify handlers all incoming message events.
func (n LocalNotifier) Notify(topic string, message []byte) error {
	log.Infof("received notification: %s: %s", topic, message)

	var genericMsg struct {
		TheirDID   string         `json:"theirdid"`
		MyDID      string         `json:"mydid"`
		GenericMsg genericChatMsg `json:"message"`
	}

	if err := json.Unmarshal(message, &genericMsg); err != nil {
		log.Errorln(err)
		return err
	}

	connection, err := n.ConnectionLookup.GetConnectionRecordByTheirDID(genericMsg.GenericMsg.SenderDID)
	if err != nil {
		log.Errorln(err)
		return err
	}

	n.RuntimeMsgs.Notification <- config.NewAppMsg(
		config.MsgTextReceived,
		model.NewTextMessage(connection.ConnectionID, genericMsg.GenericMsg.From, genericMsg.GenericMsg.Message),
	)

	return nil
}
