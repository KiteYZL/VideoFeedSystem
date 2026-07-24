package video

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type cursor struct {
	SnapshotID string `json:"snapshot_id"`
	HotOffset  int    `json:"hot_offset"`
	ColdCursor string `json:"cold_cursor"`
	ColdID     uint   `json:"cold_id"`
}

type FeedStore struct {
	redis         *redis.Client
	windowMinutes int
	snapshotTTL   time.Duration
}

func NewFeedStore(client *redis.Client, windowMinutes int, snapshotTTL time.Duration) *FeedStore {
	if windowMinutes <= 0 {
		windowMinutes = 60
	}
	return &FeedStore{redis: client, windowMinutes: windowMinutes, snapshotTTL: snapshotTTL}
}

func (f *FeedStore) Read(ctx context.Context, repo *Repository, rawCursor string, limit int) (*FeedResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	c := cursor{}
	if rawCursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(rawCursor)
		if err != nil {
			return nil, errors.New("invalid feed cursor")
		}
		if err := json.Unmarshal(data, &c); err != nil || c.SnapshotID == "" {
			return nil, errors.New("invalid feed cursor")
		}
	} else {
		c.SnapshotID = strconv.FormatInt(time.Now().UnixNano(), 10)
		if err := f.buildSnapshot(ctx, c.SnapshotID); err != nil {
			return nil, err
		}
	}
	exists, err := f.redis.Exists(ctx, HotSnapshotPrefix+c.SnapshotID).Result()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, errors.New("feed snapshot expired")
	}
	ids, err := f.redis.ZRevRange(ctx, HotSnapshotPrefix+c.SnapshotID, int64(c.HotOffset), int64(c.HotOffset+limit-1)).Result()
	if err != nil {
		return nil, err
	}
	c.HotOffset += len(ids)
	items := make([]*Video, 0, limit)
	seen := make(map[uint]struct{})
	for _, rawID := range ids {
		id, err := strconv.ParseUint(rawID, 10, 64)
		if err != nil {
			continue
		}
		item, err := repo.FindPublishedVideo(ctx, uint(id))
		if err == nil {
			items = append(items, item)
			seen[uint(id)] = struct{}{}
		}
	}
	if len(items) < limit {
		coldCursor := time.Time{}
		if c.ColdCursor != "" {
			if value, err := time.Parse(time.RFC3339Nano, c.ColdCursor); err == nil {
				coldCursor = value
			}
		}
		cold, err := repo.ListPublishedAfter(ctx, coldCursor, c.ColdID, limit*2)
		if err != nil {
			return nil, err
		}
		for _, item := range cold {
			if _, ok := seen[item.ID]; ok {
				continue
			}
			items = append(items, item)
			if item.PublishedAt != nil {
				c.ColdCursor = item.PublishedAt.Format(time.RFC3339Nano)
				c.ColdID = item.ID
			}
			if len(items) == limit {
				break
			}
		}
	}
	result := &FeedResponse{Items: items}
	if len(ids) == limit || len(items) == limit {
		data, _ := json.Marshal(c)
		result.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return result, nil
}

func (f *FeedStore) buildSnapshot(ctx context.Context, snapshotID string) error {
	keys := make([]string, 0, f.windowMinutes)
	now := time.Now().UTC().Truncate(time.Minute)
	for i := 0; i < f.windowMinutes; i++ {
		keys = append(keys, hotWindowKey(now.Add(-time.Duration(i)*time.Minute)))
	}
	destination := HotSnapshotPrefix + snapshotID
	if err := f.redis.ZUnionStore(ctx, destination, &redis.ZStore{Keys: keys, Aggregate: "SUM"}).Err(); err != nil {
		return err
	}
	return f.redis.Expire(ctx, destination, f.snapshotTTL).Err()
}

func EncodeCursor(c cursor) string {
	data, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeCursor(raw string) (cursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return cursor{}, err
	}
	var value cursor
	if err := json.Unmarshal(data, &value); err != nil {
		return cursor{}, err
	}
	return value, nil
}

func cursorString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return strings.TrimSpace(t.Format(time.RFC3339Nano))
}
