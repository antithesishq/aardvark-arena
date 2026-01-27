// Package main implements the player binary.
package main

import (
	"log"
	"os"
)

func main() {
	log.SetOutput(os.Stdout)
	log.Println("starting player...")

	// TODO: Get or generate PlayerId
	// TODO: Queue with matchmaker
	// TODO: Poll until matched
	// TODO: Connect to gameserver and play
	// TODO: Loop back to queue
}
