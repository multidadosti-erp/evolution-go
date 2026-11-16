package whatsmeow_service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	producer_interfaces "github.com/evolution-foundation/evolution-go/pkg/events/interfaces"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	instance_repository "github.com/evolution-foundation/evolution-go/pkg/instance/repository"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"github.com/patrickmn/go-cache"
	"go.mau.fi/whatsmeow/types/events"
)

type connectionTestRepository struct {
	instance_repository.InstanceRepository
	connected bool
	reason    string
}

func (r *connectionTestRepository) UpdateConnected(id string, connected bool, reason string) error {
	r.connected = connected
	r.reason = reason
	return nil
}

type connectionTestService struct {
	WhatsmeowService
	dispatched chan []byte
	restarted  chan string
}

func (s *connectionTestService) CallWebhook(instance *instance_model.Instance, queue string, data []byte) {
	s.dispatched <- data
}

func (s *connectionTestService) ReconnectClient(id string) error {
	s.restarted <- id
	return nil
}

func TestDisconnectedDoesNotRecreateClient(t *testing.T) {
	cfg := &config.Config{LogDirectory: t.TempDir()}
	loggers := logger_wrapper.NewLoggerManager(cfg)
	t.Cleanup(func() { _ = loggers.GetLogger("test").Close() })
	repository := &connectionTestRepository{connected: true}
	service := &connectionTestService{
		dispatched: make(chan []byte, 1),
		restarted:  make(chan string, 1),
	}
	instance := &instance_model.Instance{Id: "test", Token: "test-token", Connected: true, Events: "CONNECTION"}
	client := &MyClient{
		userID: "test", token: instance.Token, Instance: instance,
		config: cfg, loggerWrapper: loggers, instanceRepository: repository,
		userInfoCache: cache.New(cache.NoExpiration, 0), service: service,
	}
	client.userInfoCache.Set(instance.Token, "cached", cache.NoExpiration)

	client.myEventHandler(&events.Disconnected{})

	if repository.connected || repository.reason == "" {
		t.Fatal("disconnection was not persisted")
	}
	if _, found := client.userInfoCache.Get(instance.Token); found {
		t.Fatal("stale session cache was not cleared")
	}
	select {
	case data := <-service.dispatched:
		var payload struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Event != "Disconnected" {
			t.Fatalf("unexpected event: %s", payload.Event)
		}
	case <-time.After(time.Second):
		t.Fatal("disconnection webhook was not dispatched")
	}
	select {
	case <-service.restarted:
		t.Fatal("disconnection recreated the client instead of letting whatsmeow recover")
	case <-time.After(100 * time.Millisecond):
	}
}

type connectionTestProducer struct {
	producer_interfaces.Producer
	calls int
}

func (p *connectionTestProducer) Produce(queue string, data []byte, destination string, userID string) error {
	p.calls++
	return nil
}

func TestKeepAliveEventsDispatchAndRespectSubscriptions(t *testing.T) {
	for _, eventName := range []string{"KeepAliveTimeout", "KeepAliveRestored"} {
		t.Run(eventName, func(t *testing.T) {
			cfg := &config.Config{LogDirectory: t.TempDir()}
			loggers := logger_wrapper.NewLoggerManager(cfg)
			t.Cleanup(func() { _ = loggers.GetLogger("test").Close() })
			service := &connectionTestService{dispatched: make(chan []byte, 1)}
			instance := &instance_model.Instance{Id: "test", Webhook: "https://example.invalid/webhook"}
			client := &MyClient{
				userID: "test", Instance: instance, config: cfg,
				loggerWrapper: loggers, service: service,
			}
			lastSuccess := time.Now().UTC().Truncate(time.Second)
			if eventName == "KeepAliveTimeout" {
				client.myEventHandler(&events.KeepAliveTimeout{ErrorCount: 3, LastSuccess: lastSuccess})
			} else {
				client.myEventHandler(&events.KeepAliveRestored{})
			}
			var data []byte
			select {
			case data = <-service.dispatched:
			case <-time.After(time.Second):
				t.Fatal("keepalive event was not dispatched")
			}
			var payload struct {
				Event string
				Data  struct {
					ErrorCount  int
					LastSuccess time.Time
				}
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Event != eventName {
				t.Fatalf("got event %s, want %s", payload.Event, eventName)
			}
			if eventName == "KeepAliveTimeout" &&
				(payload.Data.ErrorCount != 3 || !payload.Data.LastSuccess.Equal(lastSuccess)) {
				t.Fatalf("unexpected timeout details: %+v", payload.Data)
			}
			producer := &connectionTestProducer{}
			backend := &whatsmeowService{loggerWrapper: loggers, webhookProducer: producer}
			for _, subscription := range []string{"MESSAGE", "CONNECTION"} {
				instance.Events = subscription
				backend.CallWebhook(instance, "test."+strings.ToLower(eventName), data)
				want := 0
				if subscription == "CONNECTION" {
					want = 1
				}
				if producer.calls != want {
					t.Fatalf("subscription %s forwarded %d events, want %d", subscription, producer.calls, want)
				}
			}
			cfg.AmqpGlobalEnabled = true
			cfg.NatsGlobalEnabled = true
			cfg.AmqpGlobalEvents = []string{"CONNECTION"}
			cfg.NatsGlobalEvents = []string{"CONNECTION"}
			amqp, nats := &connectionTestProducer{}, &connectionTestProducer{}
			backend.config = cfg
			backend.rabbitmqProducer = amqp
			backend.natsProducer = nats
			backend.SendToGlobalQueues(eventName, data, instance.Id)
			if amqp.calls != 1 || nats.calls != 1 {
				t.Fatalf("global connection subscribers missed event: AMQP=%d NATS=%d", amqp.calls, nats.calls)
			}
		})
	}
}
