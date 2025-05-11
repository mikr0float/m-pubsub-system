package subpub_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mikr0float/m-pubsub-system/subpub"
)

// НЕ ЗАБЫТЬ ПОМЕНЯТЬ ИМПОРТ!!!!!!!!!!!!!!!!!
// do go.mod!!!!!
// del else !!!!!!!! 2 times

func TestNewSubPub(t *testing.T) {
	t.Run("create subPub bus", func(t *testing.T) {
		bus := subpub.NewSubPub()
		if bus == nil {
			t.Error("expected non-nil bus")
		}
	})
}

func TestSubscribePublish(t *testing.T) {
	bus := subpub.NewSubPub()

	t.Run("Subscribe, unsubscribe and publish", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		received := false
		sub, err := bus.Subscribe("Legend", func(msg interface{}) {
			if msg != "hello" {
				t.Error("wrong message")
			} else {
				received = true
			}
			wg.Done()
		})
		if err != nil {
			t.Fatalf("subscribe failed: %v", err)
		}
		defer sub.Unsubscribe()

		err = bus.Publish("Legend", "hello")
		if err != nil {
			t.Fatalf("publish failed: %v", err)
		}

		wg.Wait()
		if !received {
			t.Error("message not received")
		}
	})
}

func TestUnsubscribe(t *testing.T) {
	bus := subpub.NewSubPub()

	t.Run("unsubscribe", func(t *testing.T) {
		var called bool

		sub, err := bus.Subscribe("unsubscribe", func(msg interface{}) {
			called = true
		})
		if err != nil {
			t.Fatalf("subscribe failed: %v", err)
		}

		sub.Unsubscribe()
		err = bus.Publish("unsubscribe", "r u stil work")
		if err != nil {
			t.Fatalf("publish failed: %v", err)
		}

		if called {
			t.Errorf("handler was called after unsubscribe")
		}
	})
}

func TestClose(t *testing.T) {

	t.Run("Graceful shutdown", func(t *testing.T) {
		bus := subpub.NewSubPub()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		received := false
		sub, err := bus.Subscribe("shutdown", func(msg interface{}) {
			if msg != "message" {
				t.Errorf("wrong message")
			} else {
				received = true
			}
			wg.Done()
		})
		if err != nil {
			t.Fatalf("subscribe failed: %v", err)
		}

		defer sub.Unsubscribe()

		err = bus.Publish("shutdown", "message")
		if err != nil {
			t.Fatalf("publish failed: %v", err)
		}

		err = bus.Close(ctx)
		if err != nil {
			t.Fatalf("close failed: %v", err)
		}
		if !received {
			t.Errorf("message is not received before close")
		}
	})

	t.Run("cancel context", func(t *testing.T) {
		bus := subpub.NewSubPub()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := bus.Close(ctx)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	})
}
