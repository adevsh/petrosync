package ws

import (
	"context"
	"testing"

	"github.com/valkey-io/valkey-go"
)

type fakePubSubReceiver struct {
	msgs []valkey.PubSubMessage
}

func (f *fakePubSubReceiver) Receive(ctx context.Context, subscribe valkey.Completed, fn func(msg valkey.PubSubMessage)) error {
	for _, m := range f.msgs {
		fn(m)
	}
	return nil
}

type fakeTextBroadcaster struct {
	msgs []string
}

func (f *fakeTextBroadcaster) BroadcastText(msg string) {
	f.msgs = append(f.msgs, msg)
}

func TestRunValkeyBridge_BroadcastsMessages(t *testing.T) {
	r := &fakePubSubReceiver{
		msgs: []valkey.PubSubMessage{
			{Channel: "ws:trip:1", Message: `{"trip_id":1}`},
			{Channel: "ws:trip:2", Message: `{"trip_id":2}`},
		},
	}
	h := &fakeTextBroadcaster{}

	var sub valkey.Completed
	if err := RunValkeyBridge(context.Background(), r, sub, h); err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(h.msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(h.msgs))
	}
	if h.msgs[0] != `{"trip_id":1}` || h.msgs[1] != `{"trip_id":2}` {
		t.Fatalf("unexpected messages: %#v", h.msgs)
	}
}
