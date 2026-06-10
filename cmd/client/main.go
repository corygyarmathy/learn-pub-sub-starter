package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	const rabbitConnString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	fmt.Println("Peril game client connected to RabbitMQ!")
	defer func() {
		if cerr := conn.Close(); cerr != nil && err == nil {
			log.Fatalf("RabbitMQ connection error: %v", cerr)
		}
	}()

	// Prompt user for username
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}

	_, _, err = pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		pubsub.Transient,
	)
	if err != nil {
		log.Fatalf("could not declare and bind queue: %v", err)
	}

	gamestate := gamelogic.NewGameState(username)
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "spawn":
			err := gamestate.CommandSpawn(words)
			if err != nil {
				log.Fatalf("could not perform command spawn: %v", err)
			}
		case "move":
			_, err := gamestate.CommandMove(words)
			if err != nil {
				log.Fatalf("could not perform command move: %v", err)
			}
			log.Println("Move successful!")
		case "status":
			gamestate.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Command unknown, please retry!")
			continue
		}
	}

}
