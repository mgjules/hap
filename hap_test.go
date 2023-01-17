package hap_test

import (
	"context"
	"testing"
	"time"

	"github.com/mgjules/hap"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAddRemoveHandlers(t *testing.T) {
	t.Parallel()

	type testEventData struct {
		*hap.Metadata
	}

	testEvent := hap.NewEvent[testEventData](zap.NewNop().Sugar())

	assert.Equal(t, 0, testEvent.NumHandlers())

	removeHandlerFn1 := testEvent.AddHandler(
		context.Background(),
		"first_handler",
		func(*testEventData) {},
	)
	assert.Equal(t, 1, testEvent.NumHandlers())

	removeHandlerFn2 := testEvent.AddBufferedHandler(
		context.Background(),
		"second_handler",
		func([]*testEventData) {},
		0,
		0,
	)
	assert.Equal(t, 2, testEvent.NumHandlers())

	removeHandlerFn1()
	assert.Equal(t, 1, testEvent.NumHandlers())

	removeHandlerFn2()
	assert.Equal(t, 0, testEvent.NumHandlers())
}

func TestSendReceive(t *testing.T) {
	t.Parallel()

	type testEventData struct {
		*hap.Metadata
		ID string
	}

	testEvent := hap.NewEvent[testEventData](zap.NewNop().Sugar())

	publisherID := "7b87b1dd-ce18-538b-a61e-0bc77b92175f"
	ctx := context.Background()
	ctx = context.WithValue(ctx, hap.ContextKeyPublisherID, publisherID)

	origData := testEventData{
		Metadata: hap.MetadataFromContext(ctx),
		ID:       "4a2d6247-cf07-5eb8-b8fd-a27c37aaecfa",
	}

	dataCh := make(chan testEventData, 1)
	removeHandlerFn := testEvent.AddHandler(
		ctx,
		"test_handler",
		func(data *testEventData) {
			if origData.ID == data.ID && data.Metadata.PublisherID == publisherID {
				dataCh <- *data
			}
		},
	)
	t.Cleanup(func() {
		removeHandlerFn()
	})

	testEvent.Trigger(ctx, origData)

	select {
	case data := <-dataCh:
		assert.Equal(t, origData, data)
	case <-time.After(time.Second):
		t.Error("no event received: timed out")
	}
}

func BenchmarkSend(b *testing.B) {
	type testEventData struct {
		*hap.Metadata
		ID string
	}

	testEvent := hap.NewEvent[testEventData](zap.NewNop().Sugar())

	ctx := context.Background()

	removeHandlerFn := testEvent.AddHandler(
		ctx,
		"test_handler",
		func(_ *testEventData) {
			// do nothing
		},
	)
	b.Cleanup(func() {
		removeHandlerFn()
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testEvent.Trigger(ctx, testEventData{
			Metadata: hap.MetadataFromContext(ctx),
			ID:       "d4ba8877-62bf-52fc-b0ce-9027dbaf00c4",
		})
	}
}
