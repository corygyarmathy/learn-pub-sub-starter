// Package pubsub consists of internal utilities for publishing and subscribing to events
package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {

	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}
	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{ContentType: "application/json", Body: data},
	)
	if err != nil {
		log.Fatalf("Failed to publish JSON data: %v", err)
	}

	return nil
}
