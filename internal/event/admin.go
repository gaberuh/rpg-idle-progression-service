package event

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// EnsureTopics cria os tópicos Kafka necessários caso ainda não existam.
// Deve ser chamado no startup antes de iniciar o producer e o worker.
func EnsureTopics(ctx context.Context, brokers []string) error {
	cl, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return fmt.Errorf("kafka admin client: %w", err)
	}
	defer cl.Close()

	topics := []struct {
		name              string
		partitions        int32
		replicationFactor int16
	}{
		{TopicProgressionEvents, 3, 2},
	}

	req := kmsg.NewCreateTopicsRequest()
	req.TimeoutMillis = 10_000
	for _, t := range topics {
		topic := kmsg.NewCreateTopicsRequestTopic()
		topic.Topic = t.name
		topic.NumPartitions = t.partitions
		topic.ReplicationFactor = t.replicationFactor
		req.Topics = append(req.Topics, topic)
	}

	resp, err := req.RequestWith(ctx, cl)
	if err != nil {
		return fmt.Errorf("kafka CreateTopics: %w", err)
	}

	for _, t := range resp.Topics {
		if t.ErrorCode == 0 {
			slog.Info("kafka topic created", "topic", t.Topic)
			continue
		}
		if t.ErrorCode == kerr.TopicAlreadyExists.Code {
			slog.Debug("kafka topic already exists", "topic", t.Topic)
			continue
		}
		return fmt.Errorf("kafka topic %q: error code %d", t.Topic, t.ErrorCode)
	}

	return nil
}
