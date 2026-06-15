package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/gaberuh/rpg-idle-progression-service/internal/event/dto"
)

const (
	TopicProgressionEvents = "progression-events"
)

type HuntProducer interface {
	PublishHuntStarted(ctx context.Context, event dto.HuntSessionStarted) error
	PublishHuntTickResolved(ctx context.Context, event dto.HuntTickResolved) error
	PublishHuntResolved(ctx context.Context, event dto.HuntSessionResolved) error
	PublishDeathOccurred(ctx context.Context, event dto.DeathOccurred) error
}

type kafkaProducer struct {
	client *kgo.Client
}

func NewHuntProducer(brokers []string) (HuntProducer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.DefaultProduceTopic(TopicProgressionEvents),
	)
	if err != nil {
		return nil, err
	}
	return &kafkaProducer{client: client}, nil
}

func (p *kafkaProducer) PublishHuntStarted(ctx context.Context, event dto.HuntSessionStarted) error {
	return p.publish(ctx, TopicProgressionEvents, event.CharacterID.String(), event)
}

func (p *kafkaProducer) PublishHuntTickResolved(ctx context.Context, event dto.HuntTickResolved) error {
	return p.publish(ctx, TopicProgressionEvents, event.CharacterID.String(), event)
}

func (p *kafkaProducer) PublishHuntResolved(ctx context.Context, event dto.HuntSessionResolved) error {
	return p.publish(ctx, TopicProgressionEvents, event.CharacterID.String(), event)
}

func (p *kafkaProducer) PublishDeathOccurred(ctx context.Context, event dto.DeathOccurred) error {
	return p.publish(ctx, TopicProgressionEvents, event.CharacterID.String(), event)
}

func (p *kafkaProducer) publish(ctx context.Context, topic, key string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		slog.Error("kafka: marshal failed", "topic", topic, "err", err)
		return err
	}

	rec := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: b,
	}

	if err := p.client.ProduceSync(ctx, rec).FirstErr(); err != nil {
		slog.Error("kafka: produce failed", "topic", topic, "key", key, "err", err)
		return err
	}

	slog.Debug("kafka: published", "topic", topic, "key", key)
	return nil
}
