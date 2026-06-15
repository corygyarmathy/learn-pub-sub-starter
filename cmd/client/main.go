package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

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
	// Create shared channel for players to publish to exchanges
	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not create publish channel: %v", err)
	}

	// Prompt user for username
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}

	gs := gamelogic.NewGameState(username)

	// Subscribe to pause queue
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+gs.Player.Username,
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatalf("could not subscribe to pause queue: %v", err)
	}

	// Subscribe to other player's move queues
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+gs.Player.Username,
		routing.ArmyMovesPrefix+".*",
		pubsub.Transient,
		handlerMove(gs, publishCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to player's move queues: %v", err)
	}

	// Subscribe to war move outcomes
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.Durable,
		handlerWar(gs, publishCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to war move outcomes: %v", err)
	}

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "spawn":
			err := gs.CommandSpawn(words)
			if err != nil {
				log.Fatalf("could not perform command spawn: %v", err)
			}
		case "move":
			move, err := gs.CommandMove(words)
			if err != nil {
				log.Fatalf("could not perform command move: %v", err)
			}

			err = pubsub.PublishJSON(
				publishCh,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+gs.Player.Username,
				move,
			)
			if err != nil {
				log.Fatalf("could not publish move: %v", err)
			}

			log.Println("Move successful!")
		case "status":
			gs.CommandStatus()
		case "spam":
			if len(words) != 2 {
				fmt.Println("Invalid number of words for the spam commands. Provide 2 words only.")
				continue
			}
			iter, err := strconv.Atoi(words[1])
			if err != nil {
				fmt.Println("Provided spam argument could not be converted into a int. Retry.")
				continue
			}
			for range iter {
				malLog := gamelogic.GetMaliciousLog()
				err := publishGameLog(publishCh, gs.GetUsername(), malLog)
				if err != nil {
					log.Fatalf("could not publish malicious game log: %v", err)
				}
			}
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

func publishGameLog(publishCh *amqp.Channel, username, msg string) error {
	return pubsub.PublishGob(
		publishCh,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+username,
		routing.GameLog{
			Username:    username,
			CurrentTime: time.Now(),
			Message:     msg,
		},
	)
}
