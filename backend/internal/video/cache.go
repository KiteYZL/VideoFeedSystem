package video

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type localVideo struct {
	id        uint
	item      *Video
	expiresAt time.Time
}

type Cache struct {
	redis    *redis.Client
	ttl      time.Duration
	redisTTL time.Duration
	capacity int
	mu       sync.Mutex
	local    map[uint]*list.Element
	order    *list.List
}

func NewCache(client *redis.Client, capacity int, localTTL, redisTTL time.Duration) *Cache {
	if capacity <= 0 {
		capacity = 1024
	}
	if redisTTL <= 0 {
		redisTTL = localTTL
	}
	return &Cache{redis: client, capacity: capacity, ttl: localTTL, redisTTL: redisTTL, local: make(map[uint]*list.Element), order: list.New()}
}

func (c *Cache) Get(ctx context.Context, id uint) (*Video, bool) {
	now := time.Now()
	c.mu.Lock()
	element, ok := c.local[id]
	if ok && now.Before(element.Value.(localVideo).expiresAt) {
		entry := element.Value.(localVideo)
		c.order.MoveToFront(element)
		c.mu.Unlock()
		return entry.item, true
	}
	if ok {
		c.order.Remove(element)
		delete(c.local, id)
	}
	c.mu.Unlock()
	if c.redis == nil {
		return nil, false
	}
	value, err := c.redis.Get(ctx, VideoCachePrefix+itoa(id)).Result()
	if err != nil {
		return nil, false
	}
	var item Video
	if json.Unmarshal([]byte(value), &item) != nil {
		return nil, false
	}
	c.putLocal(id, &item)
	return &item, true
}

func (c *Cache) Set(ctx context.Context, item *Video) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if c.redis != nil {
		if err := c.redis.Set(ctx, VideoCachePrefix+itoa(item.ID), data, c.redisTTL).Err(); err != nil {
			return err
		}
	}
	c.putLocal(item.ID, item)
	return nil
}

func (c *Cache) Invalidate(id uint) {
	c.mu.Lock()
	if element, ok := c.local[id]; ok {
		c.order.Remove(element)
		delete(c.local, id)
	}
	c.mu.Unlock()
	if c.redis != nil {
		ctx := context.Background()
		_ = c.redis.Del(ctx, VideoCachePrefix+itoa(id)).Err()
		_ = c.redis.Publish(ctx, CacheInvalidateChannel, itoa(id)).Err()
	}
}

func (c *Cache) SubscribeInvalidations(ctx context.Context) {
	if c.redis == nil {
		return
	}
	pubsub := c.redis.Subscribe(ctx, CacheInvalidateChannel)
	go func() {
		defer pubsub.Close()
		for message := range pubsub.Channel() {
			id, err := parseUint(message.Payload)
			if err != nil {
				continue
			}
			c.mu.Lock()
			if element, ok := c.local[id]; ok {
				c.order.Remove(element)
				delete(c.local, id)
			}
			c.mu.Unlock()
		}
	}()
}

func (c *Cache) putLocal(id uint, item *Video) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.local[id]; ok {
		existing.Value = localVideo{id: id, item: item, expiresAt: time.Now().Add(c.ttl)}
		c.order.MoveToFront(existing)
		return
	}
	if len(c.local) >= c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.local, oldest.Value.(localVideo).id)
		}
	}
	element := c.order.PushFront(localVideo{id: id, item: item, expiresAt: time.Now().Add(c.ttl)})
	c.local[id] = element
}

func itoa(value uint) string {
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

func parseUint(value string) (uint, error) {
	var result uint
	if value == "" {
		return 0, errors.New("empty uint")
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return 0, errors.New("invalid uint")
		}
		result = result*10 + uint(char-'0')
	}
	return result, nil
}
