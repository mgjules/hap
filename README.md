# hap

[![Go version](https://img.shields.io/github/go-mod/go-version/mgjules/hap.svg)](https://pkg.go.dev/github.com/mgjules/hap)
[![hap tests](https://github.com/mgjules/hap/actions/workflows/test.yml/badge.svg)](https://github.com/mgjules/hap/actions/workflows/test.yml)

hap, like in what's `hap`pening, is a generic event system aimed towards simplicity and performance.

## Goals

- Simple
- Fast
- Type-Safe
- Reliable

## Installation

```shell
go get -u github.com/mgjules/hap
```

## Usage example

examples/simple.go:
```go
package main

import (
	"context"
	"fmt"
	"os"
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

	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Printf("Error creating logger: %v\n", err)
		os.Exit(1)
	}
	sugaredLogger := logger.Sugar()

	// Create the event.
	userCreatedEvent := hap.NewEvent[userCreatedEventData](sugaredLogger)

	ctx := context.Background()

	received := make(chan struct{})

	// Add handlers to the event.
	// If you don't specify an id, a random uuid will be generated.
	removeHandler := userCreatedEvent.AddHandler(
		ctx,
		"",
		func(data *userCreatedEventData) {
			// Do something with the user created event data.
			sugaredLogger.Debugf("User created event: %#v", data)
			received <- struct{}{}
		},
	)
	defer removeHandler()

	// Trigger the event.
	userCreatedEvent.Trigger(ctx, userCreatedEventData{
		Metadata:  hap.MetadataFromContext(ctx),
		ID:        "4a2d6247-cf07-5eb8-b8fd-a27c37aaecfa",
		FirstName: "John",
		LastName:  "Doe",
		CreatedAt: time.Now(),
	})

	<-received
}
```

Note: You can also add buffered handlers to capture a series of events.
