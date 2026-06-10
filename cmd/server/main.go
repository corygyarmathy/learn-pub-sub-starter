package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	const rabbitConnString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	fmt.Println("Peril game server connected to RabbitMQ!")
	defer func() {
		if cerr := conn.Close(); cerr != nil && err == nil {
			log.Fatalf("RabbitMQ connection error: %v", cerr)
		}
	}()

	// wait for SIGINT (ctrl+c)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("RabbitMQ connection closed.")
}
