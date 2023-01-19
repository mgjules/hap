# hap

hap, like in what's `hap`pening, is a generic event system aimed towards simplicity and performance.

## Goals

- Simple
- Fast
- Type-Safe
- Reliable

## Installation

```shell
$ go get -u github.com/mgjules/hap
```

## Usage example

```go
package main

import (
	"context"
	"time"

	"github.com/mgjules/hap"
	"go.uber.org/zap"
)

func main() {
	type userCreatedEventData struct {
		// Embed to satisfy the necessary constraints.
		*hap.Metadata
    
		ID        string
		FirstName string
		LastName  string
		CreatedAt time.Time
	}

	logger := zap.NewNop().Sugar()
	userCreatedEvent := hap.NewEvent[userCreatedEventData](logger)

	// Add handlers to the event.
	// If you don't specify an id, a random uuid will be generated.
	userCreatedEvent.AddHandler(
		context.Background(),
		"",
		func(*userCreatedEventData) {
			// Do something with the user created event data.
		},
	)

	// Trigger the event.
	userCreatedEvent.Trigger(ctx, userCreatedEventData{
		Metadata:  hap.MetadataFromContext(ctx),
		ID:        "4a2d6247-cf07-5eb8-b8fd-a27c37aaecfa",
		FirstName: "John",
		LastName:  "Doe",
		CreatedAt: time.Now(),
	})
}
```

Note: You can also add buffered handlers to capture a series of events.
