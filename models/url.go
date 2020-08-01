package models

//URL model
type URL struct {
	ID          string `bson:"link_id"`
	OriginalURL string `bson:"original_url"`
}
