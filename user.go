package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/globalsign/mgo/bson"
)

var discordUsers = make(map[string]*discordUser)

type discordUser struct {
	userID         string
	currentGame    string
	isPlaying      bool
	startedPlaying time.Time
}

func (user *discordUser) save() {
	timeSince := time.Since(user.startedPlaying)
	query := bson.M{"id": user.userID, "game": user.currentGame}
	if db.itemExists("gamestats", query) == true { //Game already exsists
		fmt.Fprintln(out, "user.save called stat exsist")
		var currentHours stat
		db.findOne("gamestats", query, &currentHours)
		hoursSince := currentHours.Hours + timeSince.Hours()
		db.update("gamestats", query, bson.M{"$set": bson.M{"hours": hoursSince}})
	} else { //New stat
		fmt.Fprintln(out, "user.save called new stat")
		itemToInsert := stat{
			ID:     user.userID,
			Game:   user.currentGame,
			Hours:  timeSince.Hours(),
			Ignore: false,
		}
		db.insert("gamestats", itemToInsert)
	}
}

func (user *discordUser) startTracking(presence *discordgo.PresenceUpdate) {
	user.currentGame = presence.Game.Name
	user.isPlaying = true
	user.startedPlaying = time.Now()
}

func (user *discordUser) reset() {
	user.isPlaying = false
	user.startedPlaying = time.Time{}
	user.currentGame = ""
}
