package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(move gamelogic.ArmyMove) {
		// display a new prompt (> ) when the function exits
		defer fmt.Print("> ")

		gs.HandleMove(move)
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		// display a new prompt (> ) when the function exits
		defer fmt.Print("> ")

		gs.HandlePause(ps)
	}
}
