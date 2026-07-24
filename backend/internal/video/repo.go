package video

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) DB() *gorm.DB { return r.db }

func (r *Repository) CreateVideo(ctx context.Context, video *Video) error {
	return r.db.WithContext(ctx).Create(video).Error
}

func (r *Repository) FindVideo(ctx context.Context, id uint) (*Video, error) {
	var item Video
	if err := r.db.WithContext(ctx).Where("id = ? AND status <> ?", id, StatusDeleted).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) FindPublishedVideo(ctx context.Context, id uint) (*Video, error) {
	var item Video
	if err := r.db.WithContext(ctx).Where("id = ? AND status = ?", id, StatusPublished).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) FindPublishedVideoTx(tx *gorm.DB, id uint) (*Video, error) {
	var item Video
	if err := tx.Where("id = ? AND status = ?", id, StatusPublished).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) HasInbox(ctx context.Context, eventID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&InboxEvent{}).Where("event_id = ?", eventID).Count(&count).Error
	return count > 0, err
}

func (r *Repository) ListPublishedAfter(ctx context.Context, cursor time.Time, cursorID uint, limit int) ([]*Video, error) {
	items := make([]*Video, 0, limit)
	query := r.db.WithContext(ctx).Where("status = ?", StatusPublished)
	if !cursor.IsZero() {
		query = query.Where("published_at < ? OR (published_at = ? AND id < ?)", cursor, cursor, cursorID)
	}
	err := query.Order("published_at DESC, id DESC").Limit(limit).Find(&items).Error
	return items, err
}

func (r *Repository) WithTx(fn func(*gorm.DB) error) error { return r.db.Transaction(fn) }

func (r *Repository) FindByIDTx(tx *gorm.DB, id uint) (*Video, error) {
	var item Video
	if err := tx.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) SaveTx(tx *gorm.DB, item *Video) error { return tx.Save(item).Error }

func (r *Repository) AddOutboxTx(tx *gorm.DB, event *OutboxEvent) error {
	return tx.Create(event).Error
}

func (r *Repository) FindLike(ctx context.Context, videoID, userID uint) (*Like, error) {
	var item Like
	if err := r.db.WithContext(ctx).Where("video_id = ? AND user_id = ?", videoID, userID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) FindLikeTx(tx *gorm.DB, videoID, userID uint) (*Like, error) {
	var item Like
	if err := tx.Where("video_id = ? AND user_id = ?", videoID, userID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) CreateLikeTx(tx *gorm.DB, item *Like) error { return tx.Create(item).Error }

func (r *Repository) CreateViewDedupTx(tx *gorm.DB, item *ViewDedup) error {
	return tx.Create(item).Error
}

func (r *Repository) DeleteLikeTx(tx *gorm.DB, videoID, userID uint) error {
	return tx.Where("video_id = ? AND user_id = ?", videoID, userID).Delete(&Like{}).Error
}

func (r *Repository) UpdateVideoCountTx(tx *gorm.DB, videoID uint, field string, delta int64) error {
	if field != "like_count" && field != "view_count" {
		return errors.New("unsupported video count field")
	}
	return tx.Model(&Video{}).Where("id = ?", videoID).UpdateColumn(field, gorm.Expr(field+" + ?", delta)).Error
}

func (r *Repository) ListOutbox(ctx context.Context, now time.Time, limit int) ([]*OutboxEvent, error) {
	items := make([]*OutboxEvent, 0, limit)
	err := r.db.WithContext(ctx).Where("status = ? AND next_attempt_at <= ?", OutboxPending, now).Order("id ASC").Limit(limit).Find(&items).Error
	return items, err
}

func (r *Repository) MarkOutboxSent(ctx context.Context, id uint, publishedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&OutboxEvent{}).Where("id = ?", id).Updates(map[string]interface{}{"status": OutboxSent, "published_at": publishedAt}).Error
}

func (r *Repository) MarkOutboxRetry(ctx context.Context, id uint, attempts int, next time.Time, reason string, failed bool) error {
	status := OutboxPending
	if failed {
		status = OutboxFailed
	}
	return r.db.WithContext(ctx).Model(&OutboxEvent{}).Where("id = ?", id).Updates(map[string]interface{}{"status": status, "attempts": attempts, "next_attempt_at": next, "last_error": reason}).Error
}

func (r *Repository) ClaimInbox(ctx context.Context, eventID, eventType string) (bool, error) {
	var claimed bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		claimed, err = r.ClaimInboxTx(tx, eventID, eventType)
		return err
	})
	return claimed, err
}

func (r *Repository) ClaimInboxTx(tx *gorm.DB, eventID, eventType string) (bool, error) {
	item := &InboxEvent{EventID: eventID, EventType: eventType, ProcessedAt: time.Now()}
	if err := tx.Create(item).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) DeleteInbox(ctx context.Context, eventID string) error {
	return r.db.WithContext(ctx).Where("event_id = ?", eventID).Delete(&InboxEvent{}).Error
}

func (r *Repository) SetHeatValue(ctx context.Context, videoID uint, value float64) error {
	return r.db.WithContext(ctx).Model(&Video{}).Where("id = ?", videoID).Update("heat_value", value).Error
}
