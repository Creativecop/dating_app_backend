package discovery

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	defaultImpressionSource = "DISCOVERY_FEED"
	maxImpressionBatch      = 50
)

func (s *Service) CreateImpressions(ctx context.Context, viewerUserID uint64, req CreateImpressionsRequest) (*CreateImpressionsResponse, error) {
	if len(req.Items) == 0 {
		return nil, validationError("Impression items are required", map[string]any{"field": "items"})
	}
	if len(req.Items) > maxImpressionBatch {
		return nil, validationError("Too many impression items", map[string]any{"max": maxImpressionBatch})
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = defaultImpressionSource
	}
	if len(source) > 50 {
		return nil, validationError("Source must be 50 characters or less", map[string]any{"field": "source"})
	}

	seen := make(map[string]bool, len(req.Items))
	type row struct {
		candidateUserID uint64
		distanceMeters  int
		rankPosition    int
	}
	rows := make([]row, 0, len(req.Items))
	for _, item := range req.Items {
		if item.RankPosition < 1 {
			return nil, validationError("Rank position must be positive", map[string]any{"field": "rankPosition"})
		}
		key := strings.TrimSpace(item.CandidateUserUUID)
		if seen[key] {
			return nil, validationError("Candidate user UUIDs must be unique", map[string]any{"field": "candidateUserUuid"})
		}
		seen[key] = true
		snapshot, err := s.CheckTargetDiscoverable(ctx, viewerUserID, item.CandidateUserUUID, DiscoverabilityModeProfileDetail)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row{
			candidateUserID: snapshot.TargetUserID,
			distanceMeters:  snapshot.DistanceMeters,
			rankPosition:    item.RankPosition,
		})
	}

	now := time.Now().UTC()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if err := tx.Exec(`
				INSERT INTO discovery_impressions (
				  viewer_user_id,
				  candidate_user_id,
				  source,
				  distance_meters,
				  rank_position,
				  shown_at,
				  created_at
				)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, viewerUserID, row.candidateUserID, source, row.distanceMeters, row.rankPosition, now, now).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &CreateImpressionsResponse{Inserted: len(rows)}, nil
}
