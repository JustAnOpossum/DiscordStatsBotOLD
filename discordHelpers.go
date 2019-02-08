package main

import (
	"fmt"
	"image"
	"os"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/globalsign/mgo/bson"
)

type waitingMsg struct {
	msgID          string
	channelID      string
	middleMsg      string
	currentTicker  *time.Ticker
	currentSession *discordgo.Session
}

func (waiting *waitingMsg) send(channelID string) {
	var currentClock int
	clocks := [11]int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	msg, _ := waiting.currentSession.ChannelMessageSend(channelID, ":clock1: "+waiting.middleMsg+" :clock1:")
	waiting.msgID = msg.ID
	waiting.channelID = channelID
	ticker := time.NewTicker(time.Millisecond * 1000)
	waiting.currentTicker = ticker
	go func() {
		for range ticker.C {
			currentClockStr := " :clock" + strconv.Itoa(clocks[currentClock]) + ": "
			waiting.currentSession.ChannelMessageEdit(channelID, msg.ID, currentClockStr+waiting.middleMsg+currentClockStr)
			currentClock++
			if currentClock == 10 {
				currentClock = 0
			}
		}
	}()
}

func (waiting *waitingMsg) delete() {
	waiting.currentSession.ChannelMessageDelete(waiting.channelID, waiting.msgID)
	waiting.currentTicker.Stop()
}

func handleErrorInCommand(session *discordgo.Session, channelID string, err error, waitingMsg *waitingMsg) {
	session.ChannelMessageSend(channelID, "Sorry an error occured :( "+err.Error())
	ownerID := os.Getenv("OWNERID")
	ownerDM, _ := session.UserChannelCreate(ownerID)
	session.ChannelMessageSend(ownerDM.ID, err.Error())
	waitingMsg.delete()
	fmt.Printf("%+v\n", err)
}

func removeDiscordUser(session *discordgo.Session, userID string) {
	user := discordUsers[userID]
	user.mu.Lock()
	defer user.mu.Unlock()
	startingGuildID := user.mainGuildID
	for guildID, guild := range user.otherGuilds {
		for _, member := range guild.Members {
			if member.User.ID == userID {
				updateOrSave(guild.ID, user)
				user.mainGuildID = guild.ID
				delete(user.otherGuilds, guildID)
				break
			} else {
				updateOrSave(guild.ID, user)
				delete(user.otherGuilds, guildID)
			}
		}
	}

	if len(user.otherGuilds) == 0 && user.mainGuildID == startingGuildID {
		if user.isPlaying == true {
			user.save()
		}
		delete(discordUsers, userID)
	}
}

func addDiscordUser(session *discordgo.Session, newUserID, newGuildID string, isBot bool) {
	if isBot == false {
		if _, ok := discordUsers[newUserID]; ok == false {
			discordUsers[newUserID] = &discordUser{
				userID:      newUserID,
				mainGuildID: newGuildID,
				isPlaying:   false,
			}
		} else if _, ok := discordUsers[newUserID].otherGuilds[newGuildID]; ok == false {
			discordUsers[newUserID].otherGuilds = make(map[string]*discordgo.Guild)
			guildToAdd, err := session.State.Guild(newGuildID)
			if err != nil {
				panic(err)
			}
			discordUsers[newUserID].otherGuilds[newGuildID] = guildToAdd
		}
	}
}

func addDiscordGuild(session *discordgo.Session, guildID string) {
	var presenceMap = make(map[string]*discordgo.Presence)
	guildInfo, err := session.Guild(guildID)
	if err != nil {
		panic(err)
	}
	for _, presence := range guildInfo.Presences {
		userID := presence.User.ID
		if _, ok := presenceMap[userID]; ok != true {
			presenceMap[userID] = presence
		}
	}
	for _, member := range guildInfo.Members {
		if member.User.Bot == false {
			if db.itemExists("settings", bson.M{"id": member.User.ID}) == false {
				itemToInsert := setting{
					ID:              member.User.ID,
					GraphType:       "bar",
					MentionForStats: true,
				}
				db.insert("settings", itemToInsert)
			}
			userID := member.User.ID
			presence := presenceMap[userID]
			if _, ok := discordUsers[userID]; ok != true {
				var currentGame string
				var isPlaying bool
				var startedPlaying time.Time
				if presence.Game != nil {
					currentGame = presence.Game.Name
					isPlaying = true
					startedPlaying = time.Now()
				}
				discordUsers[userID] = &discordUser{
					userID:         userID,
					mainGuildID:    member.GuildID,
					currentGame:    currentGame,
					startedPlaying: startedPlaying,
					isPlaying:      isPlaying,
				}
			} else if ok := discordUsers[userID].otherGuilds[guildID]; ok == nil && guildID != discordUsers[userID].mainGuildID {
				discordUsers[userID].otherGuilds = make(map[string]*discordgo.Guild)
				guildToAdd, err := session.State.Guild(guildID)
				if err != nil {
					panic(err)
				}
				discordUsers[userID].otherGuilds[guildID] = guildToAdd
			}
		}
	}
}

func processUserImg(userID, username string, avatar *image.Image) (*discordgo.MessageSend, error) {
	var userStats []stat
	db.findAll("gamestats", bson.M{"id": userID}, &userStats)
	totalHours := db.countHours(bson.M{"id": userID})
	totalGames := db.countGames(bson.M{"id": userID})
	imgReader, err := createImage(avatar, fmt.Sprint(totalHours), strconv.Itoa(totalGames), username, "bar", userID)
	if err != nil {
		return nil, err
	}
	discordMessageSend := &discordgo.MessageSend{
		Content: "Here are your stats " + username + "!",
		Files: []*discordgo.File{
			&discordgo.File{
				Name:        userID + ".png",
				ContentType: "image/png",
				Reader:      imgReader,
			},
		},
	}
	return discordMessageSend, nil
}
