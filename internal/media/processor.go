package media

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neoscoder/aura-backend/internal/config"
	"github.com/neoscoder/aura-backend/internal/storage"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type OSCommandRunner struct{}

func (OSCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type Processor struct {
	db        *gorm.DB
	storage   storage.Provider
	cfg       config.MediaConfig
	commands  CommandRunner
	refresher DiscoveryEligibilityRefresher
}

func NewProcessor(db *gorm.DB, provider storage.Provider, cfg config.MediaConfig, commands CommandRunner) *Processor {
	if commands == nil {
		commands = OSCommandRunner{}
	}
	return &Processor{db: db, storage: provider, cfg: cfg, commands: commands}
}

func (p *Processor) SetDiscoveryEligibilityRefresher(refresher DiscoveryEligibilityRefresher) {
	p.refresher = refresher
}

func (p *Processor) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskProcessPhoto, p.ProcessPhoto)
	mux.HandleFunc(TaskProcessIntroVideo, p.ProcessIntroVideo)
}

func (p *Processor) ProcessPhoto(ctx context.Context, task *asynq.Task) error {
	payload, err := parsePayload(task)
	if err != nil {
		return err
	}
	mediaUUID, err := uuid.Parse(payload.MediaUUID)
	if err != nil {
		return err
	}

	var media UserMedia
	var original UserMediaVariant
	err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		found, err := NewRepository(tx).LockByUUID(ctx, tx, mediaUUID)
		if err != nil {
			return err
		}
		if found == nil || found.DeletedAt != nil || found.ProcessingStatus == ProcessingDeleted {
			return nil
		}
		media = *found
		orig, err := NewRepository(tx).OriginalVariant(ctx, tx, media.ID)
		if err != nil {
			return err
		}
		if orig == nil {
			return fmt.Errorf("original variant is missing")
		}
		original = *orig
		return tx.Model(&UserMedia{}).
			Where("id = ?", media.ID).
			Updates(map[string]any{
				"processing_status": ProcessingProcessing,
				"processing_error":  nil,
				"failed_at":         nil,
				"updated_at":        time.Now().UTC(),
			}).Error
	})
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Photo processing failed.")
		return err
	}
	if media.ID == 0 || original.ID == 0 {
		return nil
	}

	originalPath, err := p.storage.LocalPath(original.ObjectKey)
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Photo processing failed.")
		return err
	}

	imageValue, err := imaging.Open(originalPath, imaging.AutoOrientation(true))
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Invalid image file.")
		return err
	}

	display := imaging.Fit(imageValue, 1080, 1080, imaging.Lanczos)
	thumbnail := imaging.Fill(imageValue, 320, 320, imaging.Center, imaging.Lanczos)
	baseDir := path.Dir(original.ObjectKey)

	displayMeta, err := p.saveJPEGVariant(ctx, media.ID, VariantDisplay, baseDir+"/display.jpg", display)
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Photo processing failed.")
		return err
	}
	thumbnailMeta, err := p.saveJPEGVariant(ctx, media.ID, VariantThumbnail, baseDir+"/thumbnail.jpg", thumbnail)
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Photo processing failed.")
		return err
	}

	err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertVariant(ctx, tx, displayMeta); err != nil {
			return err
		}
		if err := upsertVariant(ctx, tx, thumbnailMeta); err != nil {
			return err
		}
		now := time.Now().UTC()
		moderation := ModerationPending
		approvedAt := any(nil)
		if p.cfg.AutoApprove {
			moderation = ModerationApproved
			approvedAt = now
		}
		return tx.Model(&UserMedia{}).
			Where("id = ?", media.ID).
			Updates(map[string]any{
				"processing_status": ProcessingReady,
				"moderation_status": moderation,
				"processed_at":      now,
				"approved_at":       approvedAt,
				"processing_error":  nil,
				"failed_at":         nil,
				"updated_at":        now,
			}).Error
	})
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Photo processing failed.")
		return err
	}

	_ = p.refreshEligibility(ctx, media.UserID)
	return nil
}

func (p *Processor) ProcessIntroVideo(ctx context.Context, task *asynq.Task) error {
	payload, err := parsePayload(task)
	if err != nil {
		return err
	}
	mediaUUID, err := uuid.Parse(payload.MediaUUID)
	if err != nil {
		return err
	}

	var media UserMedia
	var original UserMediaVariant
	err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		found, err := NewRepository(tx).LockByUUID(ctx, tx, mediaUUID)
		if err != nil {
			return err
		}
		if found == nil || found.DeletedAt != nil || found.ProcessingStatus == ProcessingDeleted {
			return nil
		}
		media = *found
		orig, err := NewRepository(tx).OriginalVariant(ctx, tx, media.ID)
		if err != nil {
			return err
		}
		if orig == nil {
			return fmt.Errorf("original variant is missing")
		}
		original = *orig
		return tx.Model(&UserMedia{}).
			Where("id = ?", media.ID).
			Updates(map[string]any{
				"processing_status": ProcessingProcessing,
				"processing_error":  nil,
				"failed_at":         nil,
				"updated_at":        time.Now().UTC(),
			}).Error
	})
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Invalid or unsupported video file.")
		return err
	}
	if media.ID == 0 || original.ID == 0 {
		return nil
	}

	originalPath, err := p.storage.LocalPath(original.ObjectKey)
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Invalid or unsupported video file.")
		return err
	}

	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	output, probeErr := p.commands.Run(
		probeCtx,
		p.cfg.FFProbePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		originalPath,
	)
	cancel()
	if probeErr != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = probeErr.Error()
		}
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, msg, "Invalid or unsupported video file.")
		return probeErr
	}

	durationFloat, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Invalid or unsupported video file.")
		return err
	}
	if durationFloat > float64(p.cfg.MaxIntroVideoSeconds)+p.cfg.VideoDurationToleranceSeconds {
		reason := "Intro video must be 30 seconds or less."
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, reason, reason)
		return fmt.Errorf("%s", reason)
	}

	baseDir := path.Dir(original.ObjectKey)
	thumbMeta, err := p.generateVideoThumbnail(ctx, media.ID, baseDir+"/thumbnail.jpg", originalPath)
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Video processing failed.")
		return err
	}
	transcodedMeta, err := p.transcodeVideo(ctx, media.ID, baseDir+"/transcoded.mp4", originalPath)
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Video processing failed.")
		return err
	}

	duration := int(durationFloat + 0.5)
	err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertVariant(ctx, tx, thumbMeta); err != nil {
			return err
		}
		if err := upsertVariant(ctx, tx, transcodedMeta); err != nil {
			return err
		}
		now := time.Now().UTC()
		moderation := ModerationPending
		approvedAt := any(nil)
		if p.cfg.AutoApprove {
			moderation = ModerationApproved
			approvedAt = now
		}
		return tx.Model(&UserMedia{}).
			Where("id = ?", media.ID).
			Updates(map[string]any{
				"processing_status": ProcessingReady,
				"moderation_status": moderation,
				"duration_seconds":  duration,
				"processed_at":      now,
				"approved_at":       approvedAt,
				"processing_error":  nil,
				"failed_at":         nil,
				"updated_at":        now,
			}).Error
	})
	if err != nil {
		_ = p.markFailed(ctx, media.ID, media.UserID, media.MediaPurpose, err.Error(), "Video processing failed.")
		return err
	}

	return nil
}

func (p *Processor) saveJPEGVariant(ctx context.Context, mediaID uint64, variantType string, objectKey string, img image.Image) (UserMediaVariant, error) {
	temp, err := p.storage.CreateTemp(ctx)
	if err != nil {
		return UserMediaVariant{}, err
	}
	if temp.File != nil {
		_ = temp.File.Close()
	}
	_ = os.Remove(temp.Path)
	temp.Path = temp.Path + ".jpg"
	if err := imaging.Save(img, temp.Path, imaging.JPEGQuality(85)); err != nil {
		cleanupTemp(temp)
		return UserMediaVariant{}, err
	}
	if err := p.storage.CommitTemp(ctx, temp, objectKey); err != nil {
		cleanupTemp(temp)
		return UserMediaVariant{}, err
	}
	stat, err := os.Stat(mustLocalPath(p.storage, objectKey))
	if err != nil {
		return UserMediaVariant{}, err
	}
	bounds := img.Bounds()
	return UserMediaVariant{
		UUID:            uuid.New(),
		MediaID:         mediaID,
		VariantType:     variantType,
		StorageProvider: p.storage.Name(),
		ObjectKey:       objectKey,
		MimeType:        stringPtr(MimeJPEG),
		SizeBytes:       int64Ptr(stat.Size()),
		Width:           intPtr(bounds.Dx()),
		Height:          intPtr(bounds.Dy()),
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func (p *Processor) generateVideoThumbnail(ctx context.Context, mediaID uint64, objectKey string, originalPath string) (UserMediaVariant, error) {
	temp, err := p.storage.CreateTemp(ctx)
	if err != nil {
		return UserMediaVariant{}, err
	}
	if temp.File != nil {
		_ = temp.File.Close()
	}
	_ = os.Remove(temp.Path)
	temp.Path = temp.Path + ".jpg"
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	output, err := p.commands.Run(runCtx, p.cfg.FFmpegPath, "-y", "-i", originalPath, "-ss", "00:00:01", "-frames:v", "1", temp.Path)
	cancel()
	if err != nil {
		cleanupTemp(temp)
		return UserMediaVariant{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := p.storage.CommitTemp(ctx, temp, objectKey); err != nil {
		cleanupTemp(temp)
		return UserMediaVariant{}, err
	}
	width, height := imageDimensions(mustLocalPath(p.storage, objectKey))
	stat, _ := os.Stat(mustLocalPath(p.storage, objectKey))
	size := int64(0)
	if stat != nil {
		size = stat.Size()
	}
	return UserMediaVariant{
		UUID:            uuid.New(),
		MediaID:         mediaID,
		VariantType:     VariantThumbnail,
		StorageProvider: p.storage.Name(),
		ObjectKey:       objectKey,
		MimeType:        stringPtr(MimeJPEG),
		SizeBytes:       int64Ptr(size),
		Width:           intPtr(width),
		Height:          intPtr(height),
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func (p *Processor) transcodeVideo(ctx context.Context, mediaID uint64, objectKey string, originalPath string) (UserMediaVariant, error) {
	temp, err := p.storage.CreateTemp(ctx)
	if err != nil {
		return UserMediaVariant{}, err
	}
	if temp.File != nil {
		_ = temp.File.Close()
	}
	_ = os.Remove(temp.Path)
	temp.Path = temp.Path + ".mp4"
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	output, err := p.commands.Run(runCtx, p.cfg.FFmpegPath, "-y", "-i", originalPath, "-c:v", "libx264", "-preset", "veryfast", "-c:a", "aac", "-movflags", "+faststart", temp.Path)
	cancel()
	if err != nil {
		cleanupTemp(temp)
		return UserMediaVariant{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := p.storage.CommitTemp(ctx, temp, objectKey); err != nil {
		cleanupTemp(temp)
		return UserMediaVariant{}, err
	}
	stat, _ := os.Stat(mustLocalPath(p.storage, objectKey))
	size := int64(0)
	if stat != nil {
		size = stat.Size()
	}
	return UserMediaVariant{
		UUID:            uuid.New(),
		MediaID:         mediaID,
		VariantType:     VariantTranscoded,
		StorageProvider: p.storage.Name(),
		ObjectKey:       objectKey,
		MimeType:        stringPtr(MimeMP4),
		SizeBytes:       int64Ptr(size),
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func (p *Processor) markFailed(ctx context.Context, mediaID uint64, userID uint64, purpose string, processingError string, rejectionReason string) error {
	if mediaID == 0 {
		return nil
	}
	now := time.Now().UTC()
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var media UserMedia
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", mediaID).First(&media).Error; err != nil {
			return err
		}
		if err := tx.Model(&UserMedia{}).
			Where("id = ?", mediaID).
			Updates(map[string]any{
				"processing_status": ProcessingFailed,
				"moderation_status": ModerationRejected,
				"processing_error":  processingError,
				"rejection_reason":  rejectionReason,
				"failed_at":         now,
				"is_primary":        false,
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}
		if media.MediaPurpose == PurposeProfilePhoto && media.IsPrimary {
			return NewService(tx, p.storage, nil, p.cfg).promotePrimaryPhoto(ctx, tx, userID)
		}
		return nil
	})
	if err == nil && purpose == PurposeProfilePhoto {
		_ = p.refreshEligibility(ctx, userID)
	}
	return err
}

func (p *Processor) refreshEligibility(ctx context.Context, userID uint64) error {
	if p.refresher == nil {
		return nil
	}
	return p.refresher.RefreshDiscoveryEligibility(ctx, userID)
}

func parsePayload(task *asynq.Task) (ProcessMediaPayload, error) {
	var payload ProcessMediaPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func upsertVariant(ctx context.Context, tx *gorm.DB, variant UserMediaVariant) error {
	return tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "media_id"}, {Name: "variant_type"}},
		DoUpdates: clause.Assignments(map[string]any{
			"storage_provider": variant.StorageProvider,
			"bucket":           variant.Bucket,
			"object_key":       variant.ObjectKey,
			"public_url":       variant.PublicURL,
			"mime_type":        variant.MimeType,
			"size_bytes":       variant.SizeBytes,
			"width":            variant.Width,
			"height":           variant.Height,
			"duration_seconds": variant.DurationSeconds,
		}),
	}).Create(&variant).Error
}

func mustLocalPath(provider storage.Provider, objectKey string) string {
	value, err := provider.LocalPath(objectKey)
	if err != nil {
		return ""
	}
	return value
}

func imageDimensions(path string) (int, int) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer file.Close()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}
