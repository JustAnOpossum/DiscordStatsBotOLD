package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

var discordUsers = make(map[string]discordUser)

type discordUser struct {
	userID         string
	currentGame    string
	isPlaying      bool
	startedPlaying time.Time
}

func (user *discordUser) Save(presence *discordgo.PresenceUpdate) {

}
