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
	mainGuildID    string
	otherGuilds    map[string]*discordgo.Guild
	currentGame    string
	isPlaying      bool
	startedPlaying time.Time
}

func updateOrSave(idToLookup string, user *discordUser) {
	if user.isPlaying == true {
		timeSince := time.Since(user.startedPlaying)
		query := bson.M{"id": idToLookup, "game": user.currentGame}
		if db.itemExists("gamestats", query) == true { //Game already exsists
			fmt.Fprintln(out, "updateOrSave called stat exsist")
			var currentHours stat
			db.findOne("gamestats", query, &currentHours)
			hoursSince := currentHours.Hours + timeSince.Hours()
			db.update("gamestats", query, bson.M{"$set": bson.M{"hours": hoursSince}})
		} else { //New stat
			fmt.Fprintln(out, "updateOrSave called new stat")
			itemToInsert := stat{
				ID:     idToLookup,
				Game:   user.currentGame,
				Hours:  timeSince.Hours(),
				Ignore: false,
			}
			db.insert("gamestats", itemToInsert)
		}
	}
}

func (user *discordUser) save() {
	updateOrSave(user.userID, user)
}

func saveGuild(user *discordUser) {
	updateOrSave(user.mainGuildID, user)
	for _, item := range user.otherGuilds {
		updateOrSave(item.ID, user)
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
