package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/globalsign/mgo/bson"
)

type keepTrackOfMsg struct {
	command   string
	id        string
	channelID string
	session   *discordgo.Session
	options   []string
	timer     *time.Timer
}

var userSettingsMap = make(map[string]*keepTrackOfMsg)

func (keepTrackOfMsg *keepTrackOfMsg) sendSettingMsg() {
	if len(keepTrackOfMsg.options) == 0 {
		keepTrackOfMsg.session.ChannelMessageSend(keepTrackOfMsg.channelID, "No Available Options For This Setting.")
		delete(userSettingsMap, keepTrackOfMsg.id)
		return
	}
	stringToSend := "**" + strings.Title(keepTrackOfMsg.command) + " Settings: (Type \"cancel\" to Cancel)**\n"
	for i, item := range keepTrackOfMsg.options {
		stringToSend += "\n" + strconv.Itoa(i+1) + ". " + strings.Title(item)
		if len(stringToSend) >= 1900 {
			stringToSend += "\u200B"
		}
	}
	messages := strings.Split(stringToSend, "\u200B")
	for i := range messages {
		keepTrackOfMsg.session.ChannelMessageSend(keepTrackOfMsg.channelID, messages[i])
	}
	keepTrackOfMsg.timer = time.NewTimer(time.Minute * 5)
	go func() {
		<-keepTrackOfMsg.timer.C
		delete(userSettingsMap, keepTrackOfMsg.id)
	}()
}

func (keepTrackOfMsg *keepTrackOfMsg) handleSettingChange(pickedOptionInt int) bool {
	if pickedOptionInt-1 > len(keepTrackOfMsg.options) {
		keepTrackOfMsg.session.ChannelMessageSend(keepTrackOfMsg.channelID, "Please Pick A Valid Option.")
		return false
	}
	pickedOption := keepTrackOfMsg.options[pickedOptionInt-1]
	switch keepTrackOfMsg.command {
	case "graph":
		db.update("settings", bson.M{"id": keepTrackOfMsg.id}, bson.M{"$set": bson.M{"graphtype": pickedOption}})
		break
	case "mention":
		parsedBool, _ := strconv.ParseBool(pickedOption)
		db.update("settings", bson.M{"id": keepTrackOfMsg.id}, bson.M{"$set": bson.M{"mentionforstats": parsedBool}})
		break
	case "hide":
		db.update("gamestats", bson.M{"id": keepTrackOfMsg.id}, bson.M{"$set": bson.M{"ignore": true}})
		break
	case "show":
		db.update("gamestats", bson.M{"id": keepTrackOfMsg.id}, bson.M{"$set": bson.M{"ignore": false}})
		break
	}
	keepTrackOfMsg.timer.Stop()
	keepTrackOfMsg.session.ChannelMessageSend(keepTrackOfMsg.channelID, "Settings Updated")
	delete(userSettingsMap, keepTrackOfMsg.id)
	return true
}

func handleSettingMsg(username, userID string) *discordgo.MessageEmbed {
	var ignoredStats []stat
	var userSettings setting
	var unignoredStats []stat
	var roleName string
	db.findAll("gamestats", bson.M{"id": userID, "ignore": true}, &ignoredStats)
	db.findOne("settings", bson.M{"id": userID}, &userSettings)
	db.findAll("gamestats", bson.M{"id": userID, "ignore": false}, &unignoredStats)
	return createMainMenu(strconv.Itoa(len(ignoredStats)), strconv.Itoa(len(unignoredStats)), userSettings.GraphType, userSettings.MentionForStats, username, roleName)
}

func createMainMenu(lengthIgnoredStats, lengthUnignoredStats, graphType string, mentionSetting bool, username, roleName string) *discordgo.MessageEmbed {
	var mentionSettingStr = "disabled"
	if mentionSetting == true {
		mentionSettingStr = "enabled"
	}
	return &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:  "**" + username + " Settings**",
				Value: "Below are the options that you can change",
			},
			&discordgo.MessageEmbedField{
				Name:  "Type \"graph\" (Current Setting: " + graphType + ")",
				Value: "This setting lets you change your graph type.",
			},
			&discordgo.MessageEmbedField{
				Name:  "Type \"hide\" (Currently Shown: " + lengthUnignoredStats + ")",
				Value: "This setting lets you hide games from your stats.",
			},
			&discordgo.MessageEmbedField{
				Name:  "Type \"show\" (Currently Hidden: " + lengthIgnoredStats + ")",
				Value: "This setting lets you show games from your stats that are ignored.",
			},
			&discordgo.MessageEmbedField{
				Name:  "Type \"mention\" (Current Setting: " + mentionSettingStr + ")",
				Value: "This lets you disable other people mentioning you to get your stats.",
			},
		},
	}
}

func createCommandMenu() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:  "***Commands***",
				Value: "Here is what you can do with the bot.\nAll commands are used by mentioning the bot",
			},
			&discordgo.MessageEmbedField{
				Name:  "\"guild\"",
				Value: "This gives you the stats for the guild.",
			},
			&discordgo.MessageEmbedField{
				Name:  "\"info\"",
				Value: "Shows the bots stats.",
			},
			&discordgo.MessageEmbedField{
				Name:  "\"settings\"",
				Value: "This lets you change your user settings",
			},
			&discordgo.MessageEmbedField{
				Name:  "Mention The Bot",
				Value: "mention the bot to get your stats.",
			},
			&discordgo.MessageEmbedField{
				Name:  "Mention Another User",
				Value: "Mention another user along with the bot and get their stats. Does not work if they have disabled it or are a bot.",
			},
		},
	}
}
