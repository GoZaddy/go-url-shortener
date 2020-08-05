package models

import "time"

//URL model
type URL struct {
	ID          string    `bson:"link_id" validate:"required"`
	OriginalURL string    `bson:"original_url" validate:"url, required"`
	ExpiresAt   time.Time `bson:"expires_at,omitempty"`
}
