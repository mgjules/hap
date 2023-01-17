package hap_test

import (
	"context"
	"testing"

	"github.com/mgjules/hap"
	"github.com/stretchr/testify/assert"
)

func TestMetadataFromContext(t *testing.T) {
	publisherID := "7b87b1dd-ce18-538b-a61e-0bc77b92175f"
	ctx := context.Background()
	ctx = context.WithValue(ctx, hap.ContextKeyPublisherID, publisherID)

	metadata := hap.MetadataFromContext(ctx)

	assert.NotEmpty(t, metadata.EventID)
	assert.Equal(t, publisherID, metadata.PublisherID)
	assert.NotEmpty(t, metadata.Time)
}
