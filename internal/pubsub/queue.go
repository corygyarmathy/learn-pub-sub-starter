package pubsub

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

// SimpleQueueType is an iota for tracking queue types
type SimpleQueueType int

const (
	Durable   SimpleQueueType = iota // Keep this queue after disconnection
	Transient                        // Do not keep this queue after disconnection
)

// DeclareAndBind enables you to declare and bind to a queue
func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // SimpleQueueType is an "enum" type I made to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("failed to create channel for connection: %v", err)
	}

	queue, err := ch.QueueDeclare(queueName,
		queueType == Durable,
		queueType == Transient,
		queueType == Transient,
		false,
		amqp.Table{"x-dead-letter-exchange": routing.ExchangePerilDeadletter},
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("failed to create queue: %v", err)
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("failed to bind queue: %v", err)
	}

	return ch, queue, nil
}
