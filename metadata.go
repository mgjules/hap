package hap

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type contextKey string

var ContextKeyPublisherID = contextKey("publisher_id")

type Metadata struct {
	// EventID is the unique identifier of the event.
	EventID string

	// PublisherID is the account which produced the event.
	PublisherID string

	// Time event was produced.
	Time time.Time
}

func MetadataFromContext(ctx context.Context) *Metadata {
	publisherID, _ := ctx.Value(ContextKeyPublisherID).(string)

	return &Metadata{
		EventID:     uuid.New().String(),
		PublisherID: publisherID,
		Time:        time.Now().UTC(),
	}
}

func (m Metadata) GetMetadata() Metadata {
	return m
}
