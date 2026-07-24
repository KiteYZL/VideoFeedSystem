package video

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusHidden    Status = "hidden"
	StatusDeleted   Status = "deleted"
)

type Video struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	AuthorID    uint       `gorm:"not null;index" json:"author_id"`
	AuthorName  string     `gorm:"type:varchar(64);not null" json:"author_name"`
	Title       string     `gorm:"type:varchar(100);not null" json:"title"`
	PlayURL     string     `gorm:"type:varchar(1024)" json:"play_url"`
	CoverURL    string     `gorm:"type:varchar(1024)" json:"cover_url"`
	Status      Status     `gorm:"type:varchar(16);not null;index" json:"status"`
	Version     uint64     `gorm:"not null;default:1" json:"version"`
	LikeCount   int64      `gorm:"not null;default:0" json:"like_count"`
	ViewCount   int64      `gorm:"not null;default:0" json:"view_count"`
	HeatValue   float64    `gorm:"not null;default:0" json:"heat_value"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type Like struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VideoID   uint      `gorm:"not null;uniqueIndex:idx_video_user" json:"video_id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_video_user" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ViewDedup struct {
	ID        uint      `gorm:"primaryKey"`
	VideoID   uint      `gorm:"not null;uniqueIndex:idx_view_day"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_view_day"`
	ViewDay   string    `gorm:"type:char(8);not null;uniqueIndex:idx_view_day"`
	CreatedAt time.Time `json:"created_at"`
}

type OutboxEvent struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	EventID          string     `gorm:"type:char(36);uniqueIndex;not null" json:"event_id"`
	EventType        string     `gorm:"type:varchar(64);index;not null" json:"event_type"`
	AggregateID      uint       `gorm:"index;not null" json:"aggregate_id"`
	AggregateVersion uint64     `gorm:"not null" json:"aggregate_version"`
	Payload          string     `gorm:"type:longtext;not null" json:"payload"`
	Status           string     `gorm:"type:varchar(16);index;not null" json:"status"`
	Attempts         int        `gorm:"not null;default:0" json:"attempts"`
	NextAttemptAt    time.Time  `gorm:"index" json:"next_attempt_at"`
	LastError        string     `gorm:"type:varchar(512)" json:"last_error,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
}

type InboxEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EventID     string    `gorm:"type:char(36);uniqueIndex;not null" json:"event_id"`
	EventType   string    `gorm:"type:varchar(64);not null" json:"event_type"`
	ProcessedAt time.Time `json:"processed_at"`
}

const (
	OutboxPending = "pending"
	OutboxSent    = "sent"
	OutboxFailed  = "failed"
)

type CreateRequest struct {
	Title    string `json:"title"`
	PlayURL  string `json:"play_url"`
	CoverURL string `json:"cover_url"`
}

type UpdateRequest struct {
	VideoID  uint    `json:"video_id" binding:"required"`
	Title    *string `json:"title"`
	PlayURL  *string `json:"play_url"`
	CoverURL *string `json:"cover_url"`
}

type VideoIDRequest struct {
	VideoID uint `json:"video_id" binding:"required"`
}

type FeedRequest struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}

type FeedResponse struct {
	Items      []*Video `json:"items"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

type LikeStatusResponse struct {
	VideoID uint `json:"video_id"`
	Liked   bool `json:"liked"`
}

type eventEnvelope struct {
	EventID          string      `json:"event_id"`
	EventType        string      `json:"event_type"`
	AggregateID      uint        `json:"aggregate_id"`
	AggregateVersion uint64      `json:"aggregate_version"`
	OccurredAt       time.Time   `json:"occurred_at"`
	Data             interface{} `json:"data"`
}

func validateTitle(title string, required bool) error {
	title = strings.TrimSpace(title)
	if required && title == "" {
		return errors.New("title is required")
	}
	if len([]rune(title)) > 100 {
		return errors.New("title must be at most 100 characters")
	}
	return nil
}

func validateMediaURL(value string, required bool) error {
	value = strings.TrimSpace(value)
	if required && value == "" {
		return errors.New("media url is required")
	}
	if value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("media url must be a valid http or https url")
	}
	return nil
}

func validatePublish(video *Video) error {
	if err := validateTitle(video.Title, true); err != nil {
		return err
	}
	if err := validateMediaURL(video.PlayURL, true); err != nil {
		return err
	}
	return validateMediaURL(video.CoverURL, true)
}

func canTransition(from, to Status) bool {
	switch from {
	case StatusDraft:
		return to == StatusPublished || to == StatusDeleted
	case StatusPublished:
		return to == StatusHidden || to == StatusDeleted
	case StatusHidden:
		return to == StatusPublished || to == StatusDeleted
	default:
		return false
	}
}
