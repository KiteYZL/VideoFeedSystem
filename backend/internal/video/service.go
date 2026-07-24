package video

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	repo     *Repository
	cache    *Cache
	feed     *FeedStore
	pageSize int
}

func NewService(repo *Repository, cache *Cache, feed *FeedStore, pageSize int) *Service {
	if pageSize <= 0 {
		pageSize = 10
	}
	return &Service{repo: repo, cache: cache, feed: feed, pageSize: pageSize}
}

func (s *Service) CreateDraft(ctx context.Context, authorID uint, authorName string, req CreateRequest) (*Video, error) {
	if err := validateTitle(req.Title, false); err != nil {
		return nil, err
	}
	if err := validateMediaURL(req.PlayURL, false); err != nil {
		return nil, err
	}
	if err := validateMediaURL(req.CoverURL, false); err != nil {
		return nil, err
	}
	item := &Video{AuthorID: authorID, AuthorName: strings.TrimSpace(authorName), Title: strings.TrimSpace(req.Title), PlayURL: strings.TrimSpace(req.PlayURL), CoverURL: strings.TrimSpace(req.CoverURL), Status: StatusDraft, Version: 1}
	if err := s.repo.CreateVideo(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) Publish(ctx context.Context, authorID, videoID uint) (*Video, error) {
	var result *Video
	err := s.repo.WithTx(func(tx *gorm.DB) error {
		item, err := s.repo.FindByIDTx(tx, videoID)
		if err != nil {
			return err
		}
		if item.AuthorID != authorID {
			return errors.New("unauthorized video owner")
		}
		if !canTransition(item.Status, StatusPublished) {
			return errors.New("invalid video status transition")
		}
		if err := validatePublish(item); err != nil {
			return err
		}
		now := time.Now().UTC()
		if item.PublishedAt == nil {
			item.PublishedAt = &now
		}
		item.Status = StatusPublished
		item.Version++
		if err := s.repo.SaveTx(tx, item); err != nil {
			return err
		}
		event, err := newEvent("video.published", item.ID, item.Version, item)
		if err != nil {
			return err
		}
		if err := s.repo.AddOutboxTx(tx, event); err != nil {
			return err
		}
		result = item
		return nil
	})
	if err == nil && s.cache != nil {
		s.cache.Invalidate(videoID)
	}
	return result, err
}

func (s *Service) Get(ctx context.Context, id uint) (*Video, error) {
	if s.cache != nil {
		if item, ok := s.cache.Get(ctx, id); ok {
			return item, nil
		}
	}
	item, err := s.repo.FindPublishedVideo(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.Set(ctx, item)
	}
	return item, nil
}

func (s *Service) Update(ctx context.Context, authorID uint, req UpdateRequest) (*Video, error) {
	var result *Video
	err := s.repo.WithTx(func(tx *gorm.DB) error {
		item, err := s.repo.FindByIDTx(tx, req.VideoID)
		if err != nil {
			return err
		}
		if item.AuthorID != authorID {
			return errors.New("unauthorized video owner")
		}
		if item.Status == StatusDeleted {
			return errors.New("deleted video cannot be updated")
		}
		if req.Title != nil {
			item.Title = strings.TrimSpace(*req.Title)
		}
		if req.PlayURL != nil {
			item.PlayURL = strings.TrimSpace(*req.PlayURL)
		}
		if req.CoverURL != nil {
			item.CoverURL = strings.TrimSpace(*req.CoverURL)
		}
		if err := validateTitle(item.Title, item.Status == StatusPublished); err != nil {
			return err
		}
		if err := validateMediaURL(item.PlayURL, item.Status == StatusPublished); err != nil {
			return err
		}
		if err := validateMediaURL(item.CoverURL, item.Status == StatusPublished); err != nil {
			return err
		}
		item.Version++
		if err := s.repo.SaveTx(tx, item); err != nil {
			return err
		}
		event, err := newEvent("video.updated", item.ID, item.Version, item)
		if err != nil {
			return err
		}
		if err := s.repo.AddOutboxTx(tx, event); err != nil {
			return err
		}
		result = item
		return nil
	})
	if err == nil && s.cache != nil {
		s.cache.Invalidate(req.VideoID)
	}
	return result, err
}

func (s *Service) ChangeStatus(ctx context.Context, authorID, videoID uint, target Status) (*Video, error) {
	var result *Video
	err := s.repo.WithTx(func(tx *gorm.DB) error {
		item, err := s.repo.FindByIDTx(tx, videoID)
		if err != nil {
			return err
		}
		if item.AuthorID != authorID {
			return errors.New("unauthorized video owner")
		}
		if !canTransition(item.Status, target) {
			return errors.New("invalid video status transition")
		}
		item.Status = target
		item.Version++
		if target == StatusDeleted {
			now := time.Now().UTC()
			item.DeletedAt = &now
		}
		if err := s.repo.SaveTx(tx, item); err != nil {
			return err
		}
		event, err := newEvent("video."+string(target), item.ID, item.Version, item)
		if err != nil {
			return err
		}
		if err := s.repo.AddOutboxTx(tx, event); err != nil {
			return err
		}
		result = item
		return nil
	})
	if err == nil && s.cache != nil {
		s.cache.Invalidate(videoID)
	}
	return result, err
}

func (s *Service) Like(ctx context.Context, userID, videoID uint, liked bool) error {
	return s.repo.WithTx(func(tx *gorm.DB) error {
		if _, err := s.repo.FindPublishedVideoTx(tx, videoID); err != nil {
			return err
		}
		current, err := s.repo.FindLikeTx(tx, videoID, userID)
		if liked {
			if err == nil && current != nil {
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := s.repo.CreateLikeTx(tx, &Like{VideoID: videoID, UserID: userID, CreatedAt: time.Now().UTC()}); err != nil {
				return err
			}
			item, err := s.repo.FindByIDTx(tx, videoID)
			if err != nil {
				return err
			}
			event, err := newEvent("like.created", videoID, item.Version, map[string]uint{"user_id": userID})
			if err != nil {
				return err
			}
			return s.repo.AddOutboxTx(tx, event)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := s.repo.DeleteLikeTx(tx, videoID, userID); err != nil {
			return err
		}
		item, err := s.repo.FindByIDTx(tx, videoID)
		if err != nil {
			return err
		}
		event, err := newEvent("like.deleted", videoID, item.Version, map[string]uint{"user_id": userID})
		if err != nil {
			return err
		}
		return s.repo.AddOutboxTx(tx, event)
	})
}

func (s *Service) GetLikeStatus(ctx context.Context, userID, videoID uint) (bool, error) {
	_, err := s.repo.FindLike(ctx, videoID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (s *Service) RecordView(ctx context.Context, userID, videoID uint) error {
	viewDay := time.Now().UTC().Format("20060102")
	err := s.repo.WithTx(func(tx *gorm.DB) error {
		if _, err := s.repo.FindPublishedVideoTx(tx, videoID); err != nil {
			return err
		}
		if err := s.repo.CreateViewDedupTx(tx, &ViewDedup{VideoID: videoID, UserID: userID, ViewDay: viewDay, CreatedAt: time.Now().UTC()}); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil
			}
			return err
		}
		item, err := s.repo.FindByIDTx(tx, videoID)
		if err != nil {
			return err
		}
		event, err := newEvent("video.viewed", videoID, item.Version, map[string]uint{"user_id": userID})
		if err != nil {
			return err
		}
		return s.repo.AddOutboxTx(tx, event)
	})
	if err == nil && s.feed != nil && s.feed.redis != nil {
		key := ViewDedupPrefix + uintString(videoID) + ":" + uintString(userID) + ":" + viewDay
		_ = s.feed.redis.Set(ctx, key, "1", 48*time.Hour).Err()
	}
	return err
}

func uintString(value uint) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[i:])
}

func (s *Service) Feed(ctx context.Context, req FeedRequest) (*FeedResponse, error) {
	if s.feed == nil {
		return nil, errors.New("feed store not initialized")
	}
	if req.Limit <= 0 {
		req.Limit = s.pageSize
	}
	return s.feed.Read(ctx, s.repo, req.Cursor, req.Limit)
}

func newEvent(eventType string, aggregateID uint, version uint64, data interface{}) (*OutboxEvent, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	eventID := hex.EncodeToString(b)
	envelope := eventEnvelope{EventID: eventID, EventType: eventType, AggregateID: aggregateID, AggregateVersion: version, OccurredAt: time.Now().UTC(), Data: data}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	return &OutboxEvent{EventID: eventID, EventType: eventType, AggregateID: aggregateID, AggregateVersion: version, Payload: string(payload), Status: OutboxPending, NextAttemptAt: time.Now()}, nil
}
