package video

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"demo/internal/config"
	"github.com/rabbitmq/amqp091-go"
)

func StartOutboxPublisher(ctx context.Context, repo *Repository, publisher *EventPublisher) {
	if publisher == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				items, err := repo.ListOutbox(ctx, now, 100)
				if err != nil {
					log.Printf("outbox query failed: %v", err)
					continue
				}
				for _, event := range items {
					if err := publisher.Publish(ctx, event); err != nil {
						attempts := event.Attempts + 1
						failed := attempts >= 5
						_ = repo.MarkOutboxRetry(ctx, event.ID, attempts, now.Add(time.Duration(attempts)*time.Second), err.Error(), failed)
						continue
					}
					_ = repo.MarkOutboxSent(ctx, event.ID, now)
				}
			}
		}
	}()
}

func StartProjectionConsumer(ctx context.Context, cfg config.RabbitConfig, repo *Repository, service *Service) error {
	conn, err := amqp091.Dial(cfg.URL)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err := ch.ExchangeDeclare(cfg.Exchange, amqp091.ExchangeTopic, true, false, false, false, nil); err != nil {
		return err
	}
	queue, err := ch.QueueDeclare(cfg.Queue, true, false, false, false, nil)
	if err != nil {
		return err
	}
	if err := ch.QueueBind(queue.Name, "#", cfg.Exchange, false, nil); err != nil {
		return err
	}
	deliveries, err := ch.Consume(queue.Name, cfg.ConsumerTag, false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		_ = ch.Close()
		_ = conn.Close()
	}()
	go func() {
		for delivery := range deliveries {
			var envelope eventEnvelope
			if err := json.Unmarshal(delivery.Body, &envelope); err != nil {
				_ = delivery.Nack(false, false)
				continue
			}
			eventType := delivery.RoutingKey
			if eventType == "" {
				eventType = envelope.EventType
			}
			event := OutboxEvent{EventID: envelope.EventID, EventType: eventType, AggregateID: envelope.AggregateID, AggregateVersion: envelope.AggregateVersion, Payload: string(delivery.Body), Status: OutboxSent}
			if err := service.ProcessEvent(ctx, event); err != nil {
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}()
	return nil
}
