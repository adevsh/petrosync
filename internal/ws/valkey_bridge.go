package ws

import (
	"context"

	"github.com/valkey-io/valkey-go"
)

type PubSubReceiver interface {
	Receive(ctx context.Context, subscribe valkey.Completed, fn func(msg valkey.PubSubMessage)) error
}

type TextBroadcaster interface {
	BroadcastText(msg string)
}

func RunValkeyBridge(ctx context.Context, receiver PubSubReceiver, subscribe valkey.Completed, hub TextBroadcaster) error {
	return receiver.Receive(ctx, subscribe, func(msg valkey.PubSubMessage) {
		hub.BroadcastText(msg.Message)
	})
}
