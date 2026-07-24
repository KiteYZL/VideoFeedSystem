package video

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"demo/internal/config"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type EventPublisher struct {
	conn     *amqp091.Connection
	channel  *amqp091.Channel
	exchange string
	confirms <-chan amqp091.Confirmation
}

func NewRedisClient(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), Password: cfg.Passwd, DB: cfg.DB})
}

func NewEventPublisher(cfg config.RabbitConfig) (*EventPublisher, error) {
	conn, err := amqp091.Dial(cfg.URL)
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := channel.ExchangeDeclare(cfg.Exchange, amqp091.ExchangeTopic, true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}
	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}
	return &EventPublisher{conn: conn, channel: channel, exchange: cfg.Exchange, confirms: channel.NotifyPublish(make(chan amqp091.Confirmation, 1))}, nil
}

func (p *EventPublisher) Publish(ctx context.Context, event *OutboxEvent) error {
	if err := p.channel.PublishWithContext(ctx, p.exchange, event.EventType, false, false, amqp091.Publishing{ContentType: "application/json", MessageId: event.EventID, Timestamp: time.Now().UTC(), Body: []byte(event.Payload), DeliveryMode: amqp091.Persistent}); err != nil {
		return err
	}
	select {
	case confirmation := <-p.confirms:
		if !confirmation.Ack {
			return errors.New("rabbitmq publisher confirm rejected")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *EventPublisher) Close() {
	if p == nil {
		return
	}
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}

func (f *FeedStore) AddPublished(ctx context.Context, item *Video) error {
	if f.redis == nil || item.PublishedAt == nil {
		return nil
	}
	return f.redis.ZAdd(ctx, GlobalTimelineKey, redis.Z{Score: float64(item.PublishedAt.Unix()), Member: strconv.FormatUint(uint64(item.ID), 10)}).Err()
}

func (f *FeedStore) RemovePublished(ctx context.Context, videoID uint) error {
	if f.redis == nil {
		return nil
	}
	return f.redis.ZRem(ctx, GlobalTimelineKey, strconv.FormatUint(uint64(videoID), 10)).Err()
}

func (f *FeedStore) ApplyPublishedState(ctx context.Context, item *Video) error {
	if f.redis == nil {
		return nil
	}
	stateKey := StateVersionPrefix + strconv.FormatUint(uint64(item.ID), 10)
	version := strconv.FormatUint(item.Version, 10)
	script := redis.NewScript(`
		local current = redis.call('GET', KEYS[2])
		if current and tonumber(current) > tonumber(ARGV[3]) then return 0 end
		if ARGV[1] == 'published' then redis.call('ZADD', KEYS[1], ARGV[2], ARGV[4]) else redis.call('ZREM', KEYS[1], ARGV[4]) end
		redis.call('SET', KEYS[2], ARGV[3])
		return 1
	`)
	score := "0"
	if item.PublishedAt != nil {
		score = strconv.FormatInt(item.PublishedAt.Unix(), 10)
	}
	_, err := script.Run(ctx, f.redis, []string{GlobalTimelineKey, stateKey}, string(item.Status), score, version, strconv.FormatUint(uint64(item.ID), 10)).Result()
	return err
}

func (f *FeedStore) AddInteraction(ctx context.Context, videoID uint, delta float64, at time.Time) error {
	if f.redis == nil {
		return nil
	}
	if err := f.redis.ZIncrBy(ctx, hotWindowKey(at), delta, strconv.FormatUint(uint64(videoID), 10)).Err(); err != nil {
		return err
	}
	return f.redis.Expire(ctx, hotWindowKey(at), time.Duration(f.windowMinutes+10)*time.Minute).Err()
}

func (f *FeedStore) AddInteractionOnce(ctx context.Context, eventID string, videoID uint, delta float64, at time.Time) error {
	if f.redis == nil {
		return nil
	}
	window := hotWindowKey(at)
	marker := ProcessedRedisEventPrefix + eventID
	script := redis.NewScript(`
		if redis.call('EXISTS', KEYS[2]) == 1 then return 0 end
		redis.call('ZINCRBY', KEYS[1], ARGV[1], ARGV[2])
		redis.call('EXPIRE', KEYS[1], ARGV[3])
		redis.call('SET', KEYS[2], '1', 'EX', ARGV[4])
		return 1
	`)
	_, err := script.Run(ctx, f.redis, []string{window, marker}, delta, strconv.FormatUint(uint64(videoID), 10), f.windowMinutes*60+600, 86400).Result()
	return err
}

func (f *FeedStore) CurrentHeat(ctx context.Context, videoID uint) (float64, error) {
	if f.redis == nil {
		return 0, errors.New("redis is not configured")
	}
	now := time.Now().UTC().Truncate(time.Minute)
	var total float64
	for i := 0; i < f.windowMinutes; i++ {
		value, err := f.redis.ZScore(ctx, hotWindowKey(now.Add(-time.Duration(i)*time.Minute)), strconv.FormatUint(uint64(videoID), 10)).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return 0, err
		}
		total += value
	}
	return total, nil
}

func (s *Service) ProcessEvent(ctx context.Context, event OutboxEvent) error {
	var envelope eventEnvelope
	if err := json.Unmarshal([]byte(event.Payload), &envelope); err != nil {
		return err
	}
	processed, err := s.repo.HasInbox(ctx, event.EventID)
	if err != nil {
		return err
	}

	switch event.EventType {
	case "video.published", "video.updated", "video.hidden", "video.deleted":
		var item Video
		if err := json.Unmarshal(rawData(envelope.Data), &item); err != nil {
			return err
		}
		if err := s.feed.ApplyPublishedState(ctx, &item); err != nil {
			return err
		}
		if s.cache != nil {
			s.cache.Invalidate(item.ID)
		}
	case "like.created":
		if err := s.feed.AddInteractionOnce(ctx, event.EventID, event.AggregateID, 5, envelope.OccurredAt); err != nil {
			return err
		}
		if s.cache != nil {
			s.cache.Invalidate(event.AggregateID)
		}
	case "like.deleted":
		if err := s.feed.AddInteractionOnce(ctx, event.EventID, event.AggregateID, -5, envelope.OccurredAt); err != nil {
			return err
		}
		if s.cache != nil {
			s.cache.Invalidate(event.AggregateID)
		}
	case "video.viewed":
		if err := s.feed.AddInteractionOnce(ctx, event.EventID, event.AggregateID, 1, envelope.OccurredAt); err != nil {
			return err
		}
		if s.cache != nil {
			s.cache.Invalidate(event.AggregateID)
		}
	}

	if !processed {
		if err := s.repo.WithTx(func(tx *gorm.DB) error {
			claimed, err := s.repo.ClaimInboxTx(tx, event.EventID, event.EventType)
			if err != nil {
				return err
			}
			if !claimed {
				return nil
			}
			switch event.EventType {
			case "like.created":
				return s.repo.UpdateVideoCountTx(tx, event.AggregateID, "like_count", 1)
			case "like.deleted":
				return s.repo.UpdateVideoCountTx(tx, event.AggregateID, "like_count", -1)
			case "video.viewed":
				return s.repo.UpdateVideoCountTx(tx, event.AggregateID, "view_count", 1)
			default:
				return nil
			}
		}); err != nil {
			return err
		}
	}

	switch event.EventType {
	case "video.published", "video.updated", "video.hidden", "video.deleted":
		return s.refreshHeat(ctx, event.AggregateID)
	case "like.created":
		return s.refreshHeat(ctx, event.AggregateID)
	case "like.deleted":
		return s.refreshHeat(ctx, event.AggregateID)
	case "video.viewed":
		return s.refreshHeat(ctx, event.AggregateID)
	default:
		return nil
	}
}

func (s *Service) refreshHeat(ctx context.Context, videoID uint) error {
	item, err := s.repo.FindVideo(ctx, videoID)
	if err != nil {
		return err
	}
	heat, err := s.feed.CurrentHeat(ctx, videoID)
	if err != nil {
		ageHours := time.Since(item.CreatedAt).Hours()
		if ageHours < 0 {
			ageHours = 0
		}
		heat = float64(item.LikeCount*5+item.ViewCount) - ageHours
	}
	return s.repo.SetHeatValue(ctx, videoID, heat)
}

func rawData(data interface{}) []byte { value, _ := json.Marshal(data); return value }
