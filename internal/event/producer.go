package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/gaberuh/rpg-idle-progression-service/internal/event/dto"
)

const (
	TopicHuntSessionResolved  = "hunt.session.resolved"
	TopicDeathOccurred        = "hunt.death.occurred"
	TopicHuntSessionCompleted = "hunt.session.completed"
)

type HuntProducer interface {
	PublishHuntResolved(ctx context.Context, event dto.HuntSessionResolved) error
	PublishDeathOccurred(ctx context.Context, event dto.DeathOccurred) error
	PublishHuntCompleted(ctx context.Context, event dto.HuntSessionCompleted) error
}

type kafkaProducer struct {
	client *kgo.Client
}

func NewHuntProducer(brokers []string) (HuntProducer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.DefaultProduceTopic(TopicHuntSessionResolved),
	)
	if err != nil {
		return nil, err
	}
	return &kafkaProducer{client: client}, nil
}

func (p *kafkaProducer) PublishHuntResolved(ctx context.Context, event dto.HuntSessionResolved) error {
	return p.publish(ctx, TopicHuntSessionResolved, event.CharacterID.String(), event)
}

func (p *kafkaProducer) PublishDeathOccurred(ctx context.Context, event dto.DeathOccurred) error {
	return p.publish(ctx, TopicDeathOccurred, event.CharacterID.String(), event)
}

func (p *kafkaProducer) PublishHuntCompleted(ctx context.Context, event dto.HuntSessionCompleted) error {
	return p.publish(ctx, TopicHuntSessionCompleted, event.CharacterID.String(), event)
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
