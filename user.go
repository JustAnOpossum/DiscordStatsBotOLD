package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

var discordUsers = make(map[string]discordUser)

type discordUser struct {
	userID         string
	currentGame    *discordgo.Game
	startedPlaying time.Time
}
