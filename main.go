package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/globalsign/mgo/bson"
)

var db *datastore
var botImgStats *imgGenFile
var out = ioutil.Discard

var dataDir = os.Getenv("DATADIR")
var gameImgDir = dataDir + "/Images/Game"
var guildBlacklists []string
var botUser *discordgo.User

func main() {
	if os.Getenv("DEBUG") == "true" {
		out = os.Stdout
	}
	envGuildBlacklists := os.Getenv("BLACKLIST")
	guildBlacklists = strings.Split(envGuildBlacklists, ",")

	session, dbStruct := setUpDB("localhost/statbot")
	db = dbStruct
	defer session.Close()

	discord, err := discordgo.New("Bot " + os.Getenv("TOKEN"))
	if err != nil {
		panic(err)
	}

	discord.AddHandler(presenceUpdate)
	discord.AddHandler(guildCreate)
	discord.AddHandler(guildDeleted)
	discord.AddHandler(memberDeleted)
	discord.AddHandler(memberAdded)
	discord.AddHandler(newMessage)

	err = discord.Open()
	if err != nil {
		panic(err)
	}

	discord.UpdateStatus(0, "@ For Stats, help")

	botUser, err = discord.User("@me")
	if err != nil {
		panic(err)
	}

	botImgStats = &imgGenFile{}
	botImgStats.load()

	fmt.Println("Bot is started")

	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-exitChan

	discord.Close()
}

func guildCreate(session *discordgo.Session, guild *discordgo.GuildCreate) {
	guildToSend, _ := session.State.Guild(guild.ID)
	addDiscordGuild(guildToSend)
}

func guildDeleted(session *discordgo.Session, deletedGuild *discordgo.GuildDelete) {
	for _, user := range discordUsers {
		if user.mainGuild == deletedGuild.ID {
			removeDiscordUser(user.userID, "")
		}
	}
}

func memberDeleted(session *discordgo.Session, deletedMember *discordgo.GuildMemberRemove) {
	if _, ok := discordUsers[deletedMember.User.ID]; ok == true {
		removeDiscordUser(deletedMember.User.ID, deletedMember.GuildID)
	}
}

func memberAdded(session *discordgo.Session, addedMember *discordgo.GuildMemberAdd) {
	addDiscordUser(addedMember.User.ID, addedMember.GuildID, addedMember.User.Bot)
}

func newMessage(session *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.Bot == true {
		return
	}
	if msg.Author.ID == botUser.ID { //Make sure bot message don't repeat
		return
	}
	if msg.GuildID == "" { //Private message handaler
		handlePrivateMessage(session, msg)
		return
	}
	var isMentioned bool
	for _, mention := range msg.Mentions {
		if mention.ID == botUser.ID {
			isMentioned = true
		}
	}
	if isMentioned == false {
		return
	}
	currentWaitMsg := &waitingMsg{
		middleMsg:      "Creating Your Image, Please Wait...",
		currentSession: session,
	}
	mentions := msg.Mentions
	switch len(mentions) {
	case 1:
		isSettingMsg, _ := regexp.MatchString("setting", msg.Content) //If setting message
		if isSettingMsg == true {
			msgToSend := handleSettingMsg(msg.Author.Username, msg.Author.ID)
			channelToSend, _ := session.UserChannelCreate(msg.Author.ID)
			session.ChannelMessageSendEmbed(channelToSend.ID, msgToSend)
			return
		}
		isHelpMsg, _ := regexp.MatchString("help", msg.Content) //If help message
		if isHelpMsg == true {
			session.ChannelMessageSendEmbed(msg.ChannelID, createCommandMenu())
			return
		}
		isGraphTypeGuild, _ := regexp.MatchString("guild", msg.Content)
		if isGraphTypeGuild == true { //Guild Handaler
			currentWaitMsg.send(msg.ChannelID)
			guildImgObj, err := handleGuildImgCreation(msg.GuildID, msg.ChannelID, session)
			if err != nil {
				handleErrorInCommand(session, msg.ChannelID, err, currentWaitMsg)
				return
			}
			session.ChannelMessageSendComplex(msg.ChannelID, guildImgObj)
			break
		}
		//Normal user handeler
		isBotInfo, _ := regexp.MatchString("info", msg.Content) //If bot info
		if isBotInfo == true {
			currentWaitMsg.send(msg.ChannelID)
			messageObj, err := processBotImg(botUser, session)
			if err != nil {
				handleErrorInCommand(session, msg.ChannelID, err, currentWaitMsg)
				return
			}
			session.ChannelMessageSendComplex(msg.ChannelID, messageObj)
			break
		}
		currentWaitMsg.send(msg.ChannelID)
		avatarURL := msg.Author.AvatarURL("512")
		userAvatar, err := loadDiscordAvatar(avatarURL)
		if err != nil {
			handleErrorInCommand(session, msg.ChannelID, err, currentWaitMsg)
			return
		}
		getTop5Img(msg.Author.ID)
		messageObj, err := processUserImg(msg.Author.ID, msg.Author.Username, userAvatar)
		if err != nil {
			handleErrorInCommand(session, msg.ChannelID, err, currentWaitMsg)
			return
		}
		session.ChannelMessageSendComplex(msg.ChannelID, messageObj)
		break
	case 2:
		var mainUser setting
		var mentionedUser setting
		db.findOne("settings", bson.M{"id": msg.Author.ID}, &mainUser)
		db.findOne("settings", bson.M{"id": mentions[0].ID}, &mentionedUser)
		if mainUser.MentionForStats == false || mentionedUser.MentionForStats == false {
			return
		}
		currentWaitMsg.send(msg.ChannelID)
		meintonedUser := mentions[0]
		avatarURL := meintonedUser.AvatarURL("512")
		userAvatar, err := loadDiscordAvatar(avatarURL)
		if err != nil {
			handleErrorInCommand(session, msg.ChannelID, err, currentWaitMsg)
			return
		}
		messageObj, err := processUserImg(meintonedUser.ID, meintonedUser.Username, userAvatar)
		if err != nil {
			handleErrorInCommand(session, msg.ChannelID, err, currentWaitMsg)
			return
		}
		session.ChannelMessageSendComplex(msg.ChannelID, messageObj)
		break
	default:
		return
	}
	currentWaitMsg.delete()
	botImgStats.increase()
}

func presenceUpdate(session *discordgo.Session, presence *discordgo.PresenceUpdate) {
	if _, ok := discordUsers[presence.User.ID]; ok == true {
		if discordUsers[presence.User.ID].mainGuild == presence.GuildID {
			if presence.Game != nil {
				if db.itemExists("iconblacklists", bson.M{"game": presence.Game.Name}) == true {
					return
				}
			}
			if discordUsers[presence.User.ID].isPlaying == true {
				if db.itemExists("iconblacklists", bson.M{"game": discordUsers[presence.User.ID].currentGame}) == true {
					return
				}
			}
		}
		handlePresenceUpdate(presence)
	}
}
