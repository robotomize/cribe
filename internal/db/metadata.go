package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
)

func NewMetadataRepository(DB *DB) *MetadataRepository {
	return &MetadataRepository{DB: DB}
}

type MetadataRepository struct {
	*DB
}

func (m *MetadataRepository) FetchByMetadata(
	ctx context.Context, videoID string, mime string, quality string,
) (Metadata, error) {
	var model Metadata
	if err := m.InTx(ctx, pgx.ReadCommitted, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT
				video_id, quality, mime, file_id, params, created_at, updated_at
			FROM
				metadata
			WHERE video_id = $1 AND quality = $2 AND mime = $3
		`, videoID, quality, mime)
		if err := row.Scan(
			&model.VideoID, &model.Quality, &model.Mime, &model.FileID, &model.Params, &model.CreatedAt, &model.UpdatedAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}

			return fmt.Errorf("transaction: %w", err)
		}

		return nil
	}); err != nil {
		return model, fmt.Errorf("fetch metadata: %w", err)
	}

	return model, nil
}

func (m *MetadataRepository) Save(ctx context.Context, model Metadata) error {
	if err := m.InTx(ctx, pgx.ReadCommitted, func(tx pgx.Tx) error {
		result, err := tx.Exec(
			ctx, `INSERT 
					INTO metadata (video_id, quality, mime, file_id, params, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7) 
					ON CONFLICT (video_id, mime, quality) 
					DO UPDATE SET file_id = $8, updated_at = $9, params = $10`,
			model.VideoID, model.Quality, model.Mime, model.FileID, model.Params, model.CreatedAt,
			model.UpdatedAt, model.FileID, model.UpdatedAt, model.Params,
		)
		if err != nil {
			return fmt.Errorf("transaction: %w", err)
		}

		if result.RowsAffected() == 0 {
			return ErrKeyConflict
		}

		return nil
	}); err != nil {
		return fmt.Errorf("insert metadata: %w", err)
	}

	return nil
}
