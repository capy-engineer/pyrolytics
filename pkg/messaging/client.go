package messaging

import (
	"context"
	"fmt"
	"log"

	"pyrolytics/config"
	"pyrolytics/internal/event_ingestor/domain"

	"github.com/bytedance/sonic"
	"github.com/nats-io/nats.go"
)

type NatsClient struct {
	Conn      *nats.Conn
	JS        nats.JetStreamContext
	StreamCfg *nats.StreamConfig
}

func NewNatsClient(cfg *config.NATSConfig) (*NatsClient, error) {
	opts := []nats.Option{
		nats.Name(cfg.ConnectionName),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			fmt.Printf("Disconnected: %v\n", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("Reconnected to %s\n", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			fmt.Printf("Connection closed: %v\n", nc.LastError())
		}),
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	maxAge, err := cfg.GetMaxAge()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("invalid max age: %w", err)
	}

	streamCfg := &nats.StreamConfig{
		Name:        cfg.StreamName,
		Description: "Solana blockchain raw events stream",
		Subjects:    cfg.StreamSubjects,
		Retention:   cfg.GetRetentionPolicy(),
		Storage:     cfg.GetStorageType(),
		MaxAge:      maxAge,
		Replicas:    cfg.Replicas,
	}

	_, err = js.AddStream(streamCfg)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	return &NatsClient{
		Conn:      nc,
		JS:        js,
		StreamCfg: streamCfg,
	}, nil
}

func (c *NatsClient) Publish(ctx context.Context, event *domain.SolanaRawEvent) error {
	eventData, err := sonic.ConfigFastest.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	subject := fmt.Sprintf("%s.%s.%s", c.StreamCfg.Name, event.DEXName, event.EventType)

	_, err = c.JS.Publish(subject, eventData)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("📤 Published event to NATS: %s (signature: %s)", subject, event.Signature[:16]+"...")

	return nil
}

func (c *NatsClient) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.JS.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	return sub, nil
}

func (c *NatsClient) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.JS.QueueSubscribe(subject, queue, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to queue subscribe: %w", err)
	}
	return sub, nil
}

func (c *NatsClient) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}
