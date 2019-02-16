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
	stringToSend := "**" + strings.Title(keepTrackOfMsg.command) + " Settings:**\n"
	for i, item := range keepTrackOfMsg.options {
		stringToSend += "\n" + strconv.Itoa(i+1) + ". " + strings.Title(item)
	}
	keepTrackOfMsg.session.ChannelMessageSend(keepTrackOfMsg.channelID, stringToSend)
	keepTrackOfMsg.timer = time.NewTimer(time.Minute * 5)
	go func() {
		<-keepTrackOfMsg.timer.C
		delete(userSettingsMap, keepTrackOfMsg.id)
	}()
}

func (keepTrackOfMsg *keepTrackOfMsg) handleSettingChange(pickedOptionInt int) {
	if pickedOptionInt-1 > len(keepTrackOfMsg.options) {
		keepTrackOfMsg.session.ChannelMessageSend(keepTrackOfMsg.channelID, "Please Pick A Valid Option.")
		return
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
}

func handleSettingMsg(session *discordgo.Session, msg *discordgo.MessageCreate, isDM bool) {
	channelToSend := msg.ChannelID
	if isDM == false {
		userSession, _ := session.UserChannelCreate(msg.Author.ID)
		channelToSend = userSession.ID
	}
	var ignoredStats []stat
	var userSettings setting
	var unignoredStats []stat
	db.findAll("gamestats", bson.M{"id": msg.Author.ID, "ignore": true}, &ignoredStats)
	db.findOne("settings", bson.M{"id": msg.Author.ID}, &userSettings)
	db.findAll("gamestats", bson.M{"id": msg.Author.ID, "ignore": false}, &unignoredStats)
	session.ChannelMessageSendEmbed(channelToSend, createMainMenu(strconv.Itoa(len(ignoredStats)), strconv.Itoa(len(unignoredStats)), userSettings.GraphType, userSettings.MentionForStats, msg.Author.Username))
}

func createMainMenu(lengthIgnoredStats, lengthUnignoredStats, graphType string, mentionSetting bool, username string) *discordgo.MessageEmbed {
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
