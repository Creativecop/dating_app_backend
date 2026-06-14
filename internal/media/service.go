package media

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/config"
	"github.com/neoscoder/aura-backend/internal/storage"
)

type DiscoveryEligibilityRefresher interface {
	RefreshDiscoveryEligibility(ctx context.Context, userID uint64) error
}

type MediaVisibilityAuthorizer interface {
	CanViewProfileMedia(ctx context.Context, viewerID uint64, ownerID uint64, mediaUUID string, variant string) bool
}

type Service struct {
	db          *gorm.DB
	repo        *Repository
	storage     storage.Provider
	queue       *asynq.Client
	cfg         config.MediaConfig
	refresher   DiscoveryEligibilityRefresher
	authorizers []MediaVisibilityAuthorizer
}

func NewService(db *gorm.DB, provider storage.Provider, queue *asynq.Client, cfg config.MediaConfig) *Service {
	return &Service{
		db:      db,
		repo:    NewRepository(db),
		storage: provider,
		queue:   queue,
		cfg:     cfg,
	}
}

func (s *Service) SetDiscoveryEligibilityRefresher(refresher DiscoveryEligibilityRefresher) {
	s.refresher = refresher
}

func (s *Service) SetMediaVisibilityAuthorizer(authorizer MediaVisibilityAuthorizer) {
	s.authorizers = nil
	if authorizer != nil {
		s.authorizers = append(s.authorizers, authorizer)
	}
}

func (s *Service) AddMediaVisibilityAuthorizer(authorizer MediaVisibilityAuthorizer) {
	if authorizer != nil {
		s.authorizers = append(s.authorizers, authorizer)
	}
}

func (s *Service) ListMine(ctx context.Context, user auth.AuthenticatedUser) (*MediaListResponse, error) {
	items, err := s.repo.ListByUser(ctx, user.UserID)
	if err != nil {
		return nil, err
	}
	return &MediaListResponse{Items: s.toMediaResponses(items)}, nil
}

func (s *Service) UploadPhoto(ctx context.Context, user auth.AuthenticatedUser, header *multipart.FileHeader, isPrimary bool) (*MediaResponse, error) {
	if header == nil {
		return nil, validationError("File is required", map[string]any{"field": "file"})
	}
	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	upload, err := streamUploadToTemp(ctx, file, s.storage, mbToBytes(s.cfg.MaxPhotoSizeMB))
	if err != nil {
		return nil, validationError("Photo is too large", map[string]any{"field": "file"})
	}
	upload.OriginalName = header.Filename
	defer cleanupTemp(upload.Temp)

	if err := validatePhotoFile(upload, mbToBytes(s.cfg.MaxPhotoSizeMB)); err != nil {
		return nil, err
	}

	mediaUUID := uuid.New()
	objectKey := photoObjectKey(user.UserUUID, mediaUUID, VariantOriginal, extensionForMime(upload.MimeType))
	if err := s.storage.CommitTemp(ctx, upload.Temp, objectKey); err != nil {
		return nil, err
	}
	upload.Temp = nil

	media, err := s.createPhotoRows(ctx, user, mediaUUID, objectKey, upload, isPrimary)
	if err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		return nil, err
	}

	task, err := NewProcessPhotoTask(media.UUID.String())
	if err == nil && s.queue != nil {
		_, err = s.queue.EnqueueContext(ctx, task, ProcessTaskOptions()...)
	}
	if err != nil || s.queue == nil {
		if s.queue == nil {
			err = fmt.Errorf("media queue client is not configured")
		}
		msg := "failed to enqueue media processing job: " + err.Error()
		_ = s.failMediaAfterCommit(ctx, media.ID, media.UserID, media.MediaPurpose, msg)
		return nil, uploadFailedError(msg)
	}

	fresh, err := s.repo.FindByUUID(ctx, user.UserID, media.UUID)
	if err != nil {
		return nil, err
	}
	response := s.toMediaResponse(*fresh)
	return &response, nil
}

func (s *Service) UploadIntroVideo(ctx context.Context, user auth.AuthenticatedUser, header *multipart.FileHeader) (*MediaResponse, error) {
	if header == nil {
		return nil, validationError("File is required", map[string]any{"field": "file"})
	}
	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	upload, err := streamUploadToTemp(ctx, file, s.storage, mbToBytes(s.cfg.MaxVideoSizeMB))
	if err != nil {
		return nil, validationError("Video is too large", map[string]any{"field": "file"})
	}
	upload.OriginalName = header.Filename
	defer cleanupTemp(upload.Temp)

	if err := validateVideoFile(upload, mbToBytes(s.cfg.MaxVideoSizeMB)); err != nil {
		return nil, err
	}

	mediaUUID := uuid.New()
	objectKey := videoObjectKey(user.UserUUID, mediaUUID, VariantOriginal, extensionForMime(upload.MimeType))
	if err := s.storage.CommitTemp(ctx, upload.Temp, objectKey); err != nil {
		return nil, err
	}
	upload.Temp = nil

	media, err := s.createIntroRows(ctx, user, mediaUUID, objectKey, upload)
	if err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		return nil, err
	}

	task, err := NewProcessIntroVideoTask(media.UUID.String())
	if err == nil && s.queue != nil {
		_, err = s.queue.EnqueueContext(ctx, task, ProcessTaskOptions()...)
	}
	if err != nil || s.queue == nil {
		if s.queue == nil {
			err = fmt.Errorf("media queue client is not configured")
		}
		msg := "failed to enqueue media processing job: " + err.Error()
		_ = s.failMediaAfterCommit(ctx, media.ID, media.UserID, media.MediaPurpose, msg)
		return nil, uploadFailedError(msg)
	}

	fresh, err := s.repo.FindByUUID(ctx, user.UserID, media.UUID)
	if err != nil {
		return nil, err
	}
	response := s.toMediaResponse(*fresh)
	return &response, nil
}

func (s *Service) SetPrimary(ctx context.Context, userID uint64, rawUUID string) (*MediaResponse, error) {
	mediaUUID, err := uuid.Parse(rawUUID)
	if err != nil {
		return nil, validationError("Media UUID is invalid", map[string]any{"mediaUuid": rawUUID})
	}

	var media UserMedia
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND uuid = ? AND deleted_at IS NULL", userID, mediaUUID).
			First(&media).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return notFoundError("Media not found")
			}
			return err
		}
		if media.MediaPurpose != PurposeProfilePhoto {
			return validationError("Only profile photos can be primary", map[string]any{"mediaUuid": rawUUID})
		}
		now := time.Now().UTC()
		if err := tx.Model(&UserMedia{}).
			Where("user_id = ? AND media_purpose = ? AND deleted_at IS NULL", userID, PurposeProfilePhoto).
			Updates(map[string]any{"is_primary": false, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&UserMedia{}).
			Where("id = ?", media.ID).
			Updates(map[string]any{"is_primary": true, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Preload("Variants").Where("id = ?", media.ID).First(&media).Error
	})
	if err != nil {
		return nil, err
	}
	_ = s.refreshEligibility(ctx, userID)
	response := s.toMediaResponse(media)
	return &response, nil
}

func (s *Service) Reorder(ctx context.Context, userID uint64, req ReorderRequest) (*MediaListResponse, error) {
	if len(req.Items) == 0 {
		return nil, validationError("Reorder items are required", map[string]any{"field": "items"})
	}
	mediaUUIDs := make([]uuid.UUID, 0, len(req.Items))
	seenUUID := make(map[uuid.UUID]bool, len(req.Items))
	seenOrder := make(map[int]bool, len(req.Items))
	orderByUUID := make(map[uuid.UUID]int, len(req.Items))

	for _, item := range req.Items {
		parsed, err := uuid.Parse(item.MediaUUID)
		if err != nil {
			return nil, validationError("Media UUID is invalid", map[string]any{"mediaUuid": item.MediaUUID})
		}
		if seenUUID[parsed] {
			return nil, validationError("Media UUIDs must be unique", map[string]any{"field": "items"})
		}
		if item.SortOrder < 1 || seenOrder[item.SortOrder] {
			return nil, validationError("Sort orders must be unique positive integers", map[string]any{"field": "sortOrder"})
		}
		seenUUID[parsed] = true
		seenOrder[item.SortOrder] = true
		mediaUUIDs = append(mediaUUIDs, parsed)
		orderByUUID[parsed] = item.SortOrder
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []UserMedia
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND uuid IN ? AND deleted_at IS NULL", userID, mediaUUIDs).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) != len(mediaUUIDs) {
			return validationError("One or more media items are invalid", map[string]any{"field": "items"})
		}
		now := time.Now().UTC()
		for _, row := range rows {
			if row.MediaPurpose != PurposeProfilePhoto {
				return validationError("Only profile photos can be reordered", map[string]any{"mediaUuid": row.UUID.String()})
			}
			if err := tx.Model(&UserMedia{}).
				Where("id = ?", row.ID).
				Updates(map[string]any{"sort_order": orderByUUID[row.UUID], "updated_at": now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.ListMine(ctx, auth.AuthenticatedUser{UserID: userID})
}

func (s *Service) Delete(ctx context.Context, userID uint64, rawUUID string) error {
	mediaUUID, err := uuid.Parse(rawUUID)
	if err != nil {
		return validationError("Media UUID is invalid", map[string]any{"mediaUuid": rawUUID})
	}
	var media UserMedia
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND uuid = ? AND deleted_at IS NULL", userID, mediaUUID).
			First(&media).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return notFoundError("Media not found")
			}
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&UserMedia{}).
			Where("id = ?", media.ID).
			Updates(map[string]any{
				"processing_status": ProcessingDeleted,
				"deleted_at":        now,
				"is_primary":        false,
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}
		if media.MediaPurpose == PurposeProfilePhoto && media.IsPrimary {
			return s.promotePrimaryPhoto(ctx, tx, userID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if media.MediaPurpose == PurposeProfilePhoto {
		_ = s.refreshEligibility(ctx, userID)
	}
	return nil
}

func (s *Service) ServeVariant(ctx context.Context, userID uint64, rawUUID string, rawVariant string) (*ServeResult, error) {
	mediaUUID, err := uuid.Parse(rawUUID)
	if err != nil {
		return nil, validationError("Media UUID is invalid", map[string]any{"mediaUuid": rawUUID})
	}
	variantType, err := normalizeVariantParam(rawVariant)
	if err != nil {
		return nil, err
	}
	media, err := s.repo.FindByUUIDAnyUser(ctx, mediaUUID)
	if err != nil {
		return nil, err
	}
	if media == nil || media.DeletedAt != nil || media.ProcessingStatus == ProcessingDeleted {
		return nil, notFoundError("Media not found")
	}
	if media.UserID != userID {
		if media.MediaPurpose != PurposeProfilePhoto ||
			media.ProcessingStatus != ProcessingReady ||
			media.ModerationStatus != ModerationApproved ||
			(variantType != VariantDisplay && variantType != VariantThumbnail) {
			return nil, forbiddenError("Media is not available")
		}
		if !s.canViewProfileMedia(ctx, userID, media.UserID, media.UUID.String(), variantType) {
			return nil, forbiddenError("Media is not available")
		}
	}
	for _, variant := range media.Variants {
		if variant.VariantType == variantType {
			mimeType := "application/octet-stream"
			if variant.MimeType != nil && *variant.MimeType != "" {
				mimeType = *variant.MimeType
			}
			localPath, _ := s.storage.LocalPath(variant.ObjectKey)
			return &ServeResult{
				MimeType:  mimeType,
				ObjectKey: variant.ObjectKey,
				LocalPath: localPath,
				AccelPath: s.cfg.NginxAccelPrefix + "/" + variant.ObjectKey,
			}, nil
		}
	}
	return nil, notFoundError("Media variant not found")
}

func (s *Service) canViewProfileMedia(ctx context.Context, viewerID uint64, ownerID uint64, mediaUUID string, variant string) bool {
	for _, authorizer := range s.authorizers {
		if authorizer != nil && authorizer.CanViewProfileMedia(ctx, viewerID, ownerID, mediaUUID, variant) {
			return true
		}
	}
	return false
}

func (s *Service) createPhotoRows(ctx context.Context, user auth.AuthenticatedUser, mediaUUID uuid.UUID, objectKey string, upload *StoredUpload, requestPrimary bool) (*UserMedia, error) {
	now := time.Now().UTC()
	var media UserMedia
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked []uint64
		if err := tx.Raw("SELECT id FROM user_media WHERE user_id = ? AND media_purpose = ? AND deleted_at IS NULL FOR UPDATE", user.UserID, PurposeProfilePhoto).Scan(&locked).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&UserMedia{}).Where("user_id = ? AND media_purpose = ? AND deleted_at IS NULL", user.UserID, PurposeProfilePhoto).Count(&count).Error; err != nil {
			return err
		}
		if count >= int64(s.cfg.MaxProfilePhotos) {
			return validationError("Maximum profile photos reached", map[string]any{"max": s.cfg.MaxProfilePhotos})
		}
		var maxSort int
		if err := tx.Model(&UserMedia{}).
			Select("COALESCE(MAX(sort_order), 0)").
			Where("user_id = ? AND media_purpose = ? AND deleted_at IS NULL", user.UserID, PurposeProfilePhoto).
			Scan(&maxSort).Error; err != nil {
			return err
		}
		isPrimary := count == 0 || requestPrimary
		if isPrimary {
			if err := tx.Model(&UserMedia{}).
				Where("user_id = ? AND media_purpose = ? AND deleted_at IS NULL", user.UserID, PurposeProfilePhoto).
				Updates(map[string]any{"is_primary": false, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		media = UserMedia{
			UUID:             mediaUUID,
			UserID:           user.UserID,
			MediaType:        MediaTypePhoto,
			MediaPurpose:     PurposeProfilePhoto,
			ProcessingStatus: ProcessingUploaded,
			ModerationStatus: ModerationPending,
			IsPrimary:        isPrimary,
			SortOrder:        maxSort + 1,
			OriginalFileName: stringPtr(upload.OriginalName),
			MimeType:         stringPtr(upload.MimeType),
			SizeBytes:        int64Ptr(upload.SizeBytes),
			Width:            intPtr(upload.Width),
			Height:           intPtr(upload.Height),
			ChecksumSHA256:   stringPtr(upload.Checksum),
			UploadedAt:       now,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := tx.Create(&media).Error; err != nil {
			return err
		}
		variant := UserMediaVariant{
			UUID:            uuid.New(),
			MediaID:         media.ID,
			VariantType:     VariantOriginal,
			StorageProvider: s.storage.Name(),
			ObjectKey:       objectKey,
			MimeType:        stringPtr(upload.MimeType),
			SizeBytes:       int64Ptr(upload.SizeBytes),
			Width:           intPtr(upload.Width),
			Height:          intPtr(upload.Height),
			CreatedAt:       now,
		}
		return tx.Create(&variant).Error
	})
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func (s *Service) createIntroRows(ctx context.Context, user auth.AuthenticatedUser, mediaUUID uuid.UUID, objectKey string, upload *StoredUpload) (*UserMedia, error) {
	now := time.Now().UTC()
	var media UserMedia
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked []uint64
		if err := tx.Raw("SELECT id FROM user_media WHERE user_id = ? AND media_purpose = ? AND deleted_at IS NULL FOR UPDATE", user.UserID, PurposeIntroVideo).Scan(&locked).Error; err != nil {
			return err
		}
		if err := tx.Model(&UserMedia{}).
			Where("user_id = ? AND media_purpose = ? AND deleted_at IS NULL", user.UserID, PurposeIntroVideo).
			Updates(map[string]any{
				"processing_status": ProcessingDeleted,
				"deleted_at":        now,
				"is_primary":        false,
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}
		media = UserMedia{
			UUID:             mediaUUID,
			UserID:           user.UserID,
			MediaType:        MediaTypeVideo,
			MediaPurpose:     PurposeIntroVideo,
			ProcessingStatus: ProcessingUploaded,
			ModerationStatus: ModerationPending,
			IsPrimary:        false,
			SortOrder:        1,
			OriginalFileName: stringPtr(upload.OriginalName),
			MimeType:         stringPtr(upload.MimeType),
			SizeBytes:        int64Ptr(upload.SizeBytes),
			ChecksumSHA256:   stringPtr(upload.Checksum),
			UploadedAt:       now,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := tx.Create(&media).Error; err != nil {
			return err
		}
		variant := UserMediaVariant{
			UUID:            uuid.New(),
			MediaID:         media.ID,
			VariantType:     VariantOriginal,
			StorageProvider: s.storage.Name(),
			ObjectKey:       objectKey,
			MimeType:        stringPtr(upload.MimeType),
			SizeBytes:       int64Ptr(upload.SizeBytes),
			CreatedAt:       now,
		}
		return tx.Create(&variant).Error
	})
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func (s *Service) failMediaAfterCommit(ctx context.Context, mediaID uint64, userID uint64, purpose string, message string) error {
	now := time.Now().UTC()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var media UserMedia
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", mediaID).First(&media).Error; err != nil {
			return err
		}
		if err := tx.Model(&UserMedia{}).
			Where("id = ?", mediaID).
			Updates(map[string]any{
				"processing_status": ProcessingFailed,
				"moderation_status": ModerationRejected,
				"processing_error":  message,
				"failed_at":         now,
				"is_primary":        false,
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}
		if media.MediaPurpose == PurposeProfilePhoto && media.IsPrimary {
			return s.promotePrimaryPhoto(ctx, tx, userID)
		}
		return nil
	})
	if err == nil && purpose == PurposeProfilePhoto {
		_ = s.refreshEligibility(ctx, userID)
	}
	return err
}

func (s *Service) promotePrimaryPhoto(ctx context.Context, tx *gorm.DB, userID uint64) error {
	var next UserMedia
	err := tx.WithContext(ctx).
		Where("user_id = ? AND media_purpose = ? AND processing_status = ? AND moderation_status = ? AND deleted_at IS NULL", userID, PurposeProfilePhoto, ProcessingReady, ModerationApproved).
		Order("sort_order ASC").
		Order("uploaded_at ASC").
		First(&next).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return tx.Model(&UserMedia{}).Where("id = ?", next.ID).Update("is_primary", true).Error
}

func (s *Service) refreshEligibility(ctx context.Context, userID uint64) error {
	if s.refresher == nil {
		return nil
	}
	return s.refresher.RefreshDiscoveryEligibility(ctx, userID)
}

func (s *Service) toMediaResponses(items []UserMedia) []MediaResponse {
	responses := make([]MediaResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, s.toMediaResponse(item))
	}
	return responses
}

func (s *Service) toMediaResponse(item UserMedia) MediaResponse {
	variants := make([]MediaVariantResponse, 0, len(item.Variants))
	for _, variant := range item.Variants {
		variants = append(variants, MediaVariantResponse{
			Type:     variant.VariantType,
			URL:      s.cfg.BaseURL + "/" + item.UUID.String() + "/" + routeVariantName(variant.VariantType),
			MimeType: variant.MimeType,
			Width:    variant.Width,
			Height:   variant.Height,
		})
	}
	return MediaResponse{
		MediaUUID:        item.UUID.String(),
		MediaType:        item.MediaType,
		MediaPurpose:     item.MediaPurpose,
		ProcessingStatus: item.ProcessingStatus,
		ModerationStatus: item.ModerationStatus,
		IsPrimary:        item.IsPrimary,
		SortOrder:        item.SortOrder,
		OriginalFileName: item.OriginalFileName,
		MimeType:         item.MimeType,
		SizeBytes:        item.SizeBytes,
		Width:            item.Width,
		Height:           item.Height,
		DurationSeconds:  item.DurationSeconds,
		ProcessingError:  item.ProcessingError,
		RejectionReason:  item.RejectionReason,
		Variants:         variants,
	}
}

func photoObjectKey(userUUID uuid.UUID, mediaUUID uuid.UUID, variant string, ext string) string {
	return "users/" + userUUID.String() + "/profile/photos/" + mediaUUID.String() + "/" + strings.ToLower(variant) + ext
}

func videoObjectKey(userUUID uuid.UUID, mediaUUID uuid.UUID, variant string, ext string) string {
	return "users/" + userUUID.String() + "/profile/videos/" + mediaUUID.String() + "/" + strings.ToLower(variant) + ext
}

func routeVariantName(variantType string) string {
	switch variantType {
	case VariantOriginal:
		return "original"
	case VariantDisplay:
		return "display"
	case VariantThumbnail:
		return "thumbnail"
	case VariantTranscoded:
		return "transcoded"
	default:
		return strings.ToLower(variantType)
	}
}

func mbToBytes(value int) int64 {
	return int64(value) * 1024 * 1024
}

func cleanupTemp(temp *storage.TempFile) {
	if temp == nil || temp.Path == "" {
		return
	}
	if temp.File != nil {
		_ = temp.File.Close()
	}
	_ = os.Remove(temp.Path)
}

func stringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func intPtr(value int) *int {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func AsServiceError(err error) (*ServiceError, bool) {
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr, true
	}
	return nil, false
}

func HTTPStatus(err error) int {
	if serviceErr, ok := AsServiceError(err); ok {
		return serviceErr.Status
	}
	return http.StatusInternalServerError
}
