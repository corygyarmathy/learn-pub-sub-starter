package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// AckType is an iota for tracking acknowledge types to messages
type AckType int

const (
	Ack         AckType = iota //  Acknowledge the message
	NackRequeue                //  Nack the message, try again
	NackDiscard                //  Nack the meaggage, do not try again
)

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	// Make sure that the given queue exists and is bound to the exchange
	ch, queue, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		queueType,
	)
	if err != nil {
		return fmt.Errorf("could not declare and bind queue: %v", err)
	}

	msgs, err := ch.Consume(
		queue.Name, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("could not consume channel: %v", err)
	}

	// Concurrently receive channel messages
	go func() {
		// Close channel on exit
		defer func() {
			if cerr := ch.Close(); cerr != nil && err == nil {
				log.Fatalf("failed to close channel: %v", cerr)
			}
		}()

		// Unmarshal, handle, and acknowledge each received message
		for msg := range msgs {
			target, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}

			switch handler(target) {
			case Ack:
				err = msg.Ack(false)
				if err != nil {
					fmt.Printf("could not acknowledge message: %v\n", err)
					continue
				}
				log.Println("Ack'd message.")
			case NackRequeue:
				err = msg.Nack(false, true)
				if err != nil {
					fmt.Printf("could not nack and requeue message: %v\n", err)
					continue
				}
				log.Println("Nack'd message. Requeuing.")
			case NackDiscard:
				err = msg.Nack(false, false)
				if err != nil {
					fmt.Printf("could not nack and discard message: %v\n", err)
					continue
				}
				log.Println("Nack'd message. Discarding.")
			}
		}
	}()

	return nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	unmarshaller := func(data []byte) (T, error) {
		var target T
		err := json.Unmarshal(data, &target)
		return target, err
	}

	err := subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)
	if err != nil {
		return fmt.Errorf("could not subscribe to JSON: %v", err)
	}

	return nil
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	unmarshaller := func(data []byte) (T, error) {
		buffer := bytes.NewBuffer(data)
		decoder := gob.NewDecoder(buffer)

		var target T
		err := decoder.Decode(&target)
		return target, err
	}

	err := subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)
	if err != nil {
		return fmt.Errorf("could not subscribe to Gob: %v", err)
	}

	return nil
}
