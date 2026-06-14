package media

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithDB(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByUser(ctx context.Context, userID uint64) ([]UserMedia, error) {
	var rows []UserMedia
	err := r.db.WithContext(ctx).
		Preload("Variants").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("media_purpose ASC").
		Order("sort_order ASC").
		Order("uploaded_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *Repository) FindByUUID(ctx context.Context, userID uint64, mediaUUID uuid.UUID) (*UserMedia, error) {
	var row UserMedia
	err := r.db.WithContext(ctx).
		Preload("Variants").
		Where("user_id = ? AND uuid = ?", userID, mediaUUID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *Repository) FindByUUIDAnyUser(ctx context.Context, mediaUUID uuid.UUID) (*UserMedia, error) {
	var row UserMedia
	err := r.db.WithContext(ctx).
		Preload("Variants").
		Where("uuid = ?", mediaUUID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *Repository) LockByUUID(ctx context.Context, tx *gorm.DB, mediaUUID uuid.UUID) (*UserMedia, error) {
	var row UserMedia
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uuid = ?", mediaUUID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *Repository) OriginalVariant(ctx context.Context, tx *gorm.DB, mediaID uint64) (*UserMediaVariant, error) {
	var row UserMediaVariant
	err := tx.WithContext(ctx).
		Where("media_id = ? AND variant_type = ?", mediaID, VariantOriginal).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}
