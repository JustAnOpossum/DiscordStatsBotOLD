package main

import (
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

type waitingMsg struct {
	msgID          string
	channelID      string
	middleMsg      string
	currentTicker  *time.Ticker
	currentSession *discordgo.Session
	stopChan       chan bool
	err            bool
}

type imgGenFile struct {
	numTimes int
	mutex    sync.Mutex
}

func (imgGenFile *imgGenFile) load() {
	imgGenFile.mutex.Lock()
	defer imgGenFile.mutex.Unlock()
	filePath := path.Join(dataDir, "botImg.txt")
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		imgGenFile.numTimes = 0
		return
	}
	parsedInt, _ := strconv.Atoi(string(file))
	imgGenFile.numTimes = parsedInt
}

func (imgGenFile *imgGenFile) increase() {
	imgGenFile.mutex.Lock()
	defer imgGenFile.mutex.Unlock()
	filePath := path.Join(dataDir, "botImg.txt")
	imgGenFile.numTimes++
	ioutil.WriteFile(filePath, []byte(strconv.Itoa(imgGenFile.numTimes)), 0644)
}

func (waiting *waitingMsg) send(channelID string) {
	var currentClock int
	clocks := [11]int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	msg, err := waiting.currentSession.ChannelMessageSend(channelID, ":clock1: "+waiting.middleMsg+" :clock1:")
	if err != nil {
		waiting.err = true
		return
	}
	waiting.msgID = msg.ID
	waiting.channelID = channelID
	ticker := time.NewTicker(time.Millisecond * 1000)
	waiting.currentTicker = ticker
	waiting.stopChan = make(chan bool, 1)
	go func() {
		for {
			select {
			case <-ticker.C:
				{
					currentClockStr := " :clock" + strconv.Itoa(clocks[currentClock]) + ": "
					waiting.currentSession.ChannelMessageEdit(channelID, msg.ID, currentClockStr+waiting.middleMsg+currentClockStr)
					currentClock++
					if currentClock == 10 {
						waiting.currentTicker.Stop()
						waiting.stopChan <- true
					}
				}
			case <-waiting.stopChan:
				{
					return
				}
			}
		}
	}()
}

func (waiting *waitingMsg) delete() {
	if waiting.err == true {
		return
	}
	waiting.currentSession.ChannelMessageDelete(waiting.channelID, waiting.msgID)
	waiting.currentTicker.Stop()
	waiting.stopChan <- true
}

func handleErrorInCommand(session *discordgo.Session, channelID string, err error, waitingMsg *waitingMsg) {
	session.ChannelMessageSend(channelID, "Sorry an error occured :( "+err.Error())
	ownerID := os.Getenv("OWNERID")
	ownerDM, _ := session.UserChannelCreate(ownerID)
	session.ChannelMessageSend(ownerDM.ID, err.Error())
	waitingMsg.delete()
	fmt.Printf("%+v\n", err)
}

func removeDiscordUser(userID, deletedGuildID string) {
	user := discordUsers[userID]
	otherGuilds := user.otherGuilds
	user.mu.Lock()
	defer user.mu.Unlock()

	if user.mainGuild == deletedGuildID { //If main guild is deleted
		if len(otherGuilds) == 0 { //No other guilds left
			if user.isPlaying == true {
				user.save()
				updateOrSave(user.mainGuild, user)
			}
			delete(discordUsers, user.userID)
			return
		}
		for _, item := range otherGuilds {
			user.mainGuild = item
			break
		}
	}
	updateOrSave(deletedGuildID, user)
	delete(otherGuilds, deletedGuildID)
}

func addDiscordUser(newUserID, newGuildID string, isBot bool) {
	for i := range guildBlacklists {
		if guildBlacklists[i] == newGuildID {
			return
		}
	}
	if isBot == true {
		return
	}
	if _, ok := discordUsers[newUserID]; ok == false {
		itemToInsert := setting{
			ID:              newUserID,
			GraphType:       "bar",
			MentionForStats: true,
		}
		db.insert("settings", itemToInsert)

		discordUsers[newUserID] = &discordUser{
			userID:      newUserID,
			mainGuild:   newGuildID,
			isPlaying:   false,
			otherGuilds: make(map[string]string),
		}
	} else if _, ok := discordUsers[newUserID].otherGuilds[newGuildID]; ok == false {
		discordUsers[newUserID].otherGuilds[newGuildID] = newGuildID
	}
}

func addDiscordGuild(guildInfo *discordgo.Guild) {
	for i := range guildBlacklists {
		if guildBlacklists[i] == guildInfo.ID {
			return
		}
	}
	var presenceMap = make(map[string]*discordgo.Presence)
	for _, presence := range guildInfo.Presences {
		userID := presence.User.ID
		if _, ok := presenceMap[userID]; ok != true {
			presenceMap[userID] = presence
		}
	}
	for _, member := range guildInfo.Members {
		if member.User.Bot == true {
			continue
		}
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
			if presence == nil {
				discordUsers[userID] = &discordUser{
					userID:      userID,
					mainGuild:   guildInfo.ID,
					otherGuilds: make(map[string]string),
				}
				continue
			}
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
				mainGuild:      guildInfo.ID,
				currentGame:    currentGame,
				startedPlaying: startedPlaying,
				isPlaying:      isPlaying,
				otherGuilds:    make(map[string]string),
			}
		} else if ok := discordUsers[userID].otherGuilds[guildInfo.ID]; ok == "" && guildInfo.ID != discordUsers[userID].mainGuild {
			discordUsers[userID].mu.Lock()
			defer discordUsers[userID].mu.Unlock()
			discordUsers[userID].otherGuilds[guildInfo.ID] = guildInfo.ID
		}
	}
}

func processUserImg(userID, username string, avatar *image.Image) (*discordgo.MessageSend, error) {
	var userStats []stat
	db.findAll("gamestats", bson.M{"id": userID}, &userStats)
	totalHours := db.countHours(bson.M{"id": userID})
	totalGames := db.countGames(bson.M{"id": userID})
	var userSetting setting
	db.findOne("settings", bson.M{"id": userID}, &userSetting)
	imgReader, err := createImage(avatar, fmt.Sprint(totalHours), strconv.Itoa(totalGames), username, userSetting.GraphType, userID)
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

func processBotImg(user *discordgo.User, session *discordgo.Session) (*discordgo.MessageSend, error) {
	avatar, err := loadDiscordAvatar(user.AvatarURL("512"))
	if err != nil {
		return nil, errors.Wrap(err, "Loading bot avatar")
	}
	var botStats []stat
	var botGames []icon
	db.findAll("gamestats", bson.M{}, &botStats)
	db.findAll("gameicons", bson.M{}, &botGames)
	totalStats := strconv.Itoa(len(botStats))
	totalGames := db.countGames(bson.M{})
	totalImgGen := botImgStats.numTimes
	totalServers := strconv.Itoa(len(session.State.Guilds))
	totalUsers := strconv.Itoa(len(discordUsers))
	imgReader, err := createBotImage(avatar, user.Username, totalStats, strconv.Itoa(totalGames), strconv.Itoa(totalImgGen), totalServers, totalUsers)
	if err != nil {
		return nil, errors.Wrap(err, "Creating bot img")
	}
	discordMessageSend := &discordgo.MessageSend{
		Content: "Here are my stats!",
		Files: []*discordgo.File{
			&discordgo.File{
				Name:        user.ID + ".png",
				ContentType: "image/png",
				Reader:      imgReader,
			},
		},
	}
	return discordMessageSend, nil
}

func handlePresenceUpdate(presence *discordgo.PresenceUpdate) {
	game := presence.Game
	user := discordUsers[presence.User.ID]
	user.mu.Lock()
	defer user.mu.Unlock()
	if game != nil { //Started Playing Game
		if game.Name != user.currentGame {
			fmt.Fprintln(out, "Started Playing Game "+game.Name)
			if user.isPlaying == true { //Switching from other game
				fmt.Fprintln(out, "Switching From Other Game "+user.currentGame)
				user.save()
				saveGuild(user)
				user.reset()
				user.startTracking(presence)
			} else { //Not currently playing game
				fmt.Fprintln(out, "Not Playing Any Game")
				user.startTracking(presence)
			}
		}
	} else { //Stopped Playing Game
		if user.currentGame != "" {
			fmt.Fprintln(out, "Stopped Playing Game")
			user.save()
			saveGuild(user)
			user.reset()
		}
	}
}

func handlePrivateMessage(session *discordgo.Session, msg *discordgo.MessageCreate) {
	if _, ok := userSettingsMap[msg.Author.ID]; ok == true {
		if strings.ToLower(msg.Content) == "cancel" {
			delete(userSettingsMap, msg.Author.ID)
			session.ChannelMessageSend(msg.ChannelID, "Cancelled")
			return
		}
		pickedOption, err := strconv.Atoi(msg.Content)
		if err != nil {
			session.ChannelMessageSend(msg.ChannelID, "Please Enter A Valid Option.")
			return
		}
		didComplete := userSettingsMap[msg.Author.ID].handleSettingChange(pickedOption)
		if didComplete == false {
			return
		}
		msgToSend := handleSettingMsg(msg.Author.Username, msg.Author.ID)
		session.ChannelMessageSendEmbed(msg.ChannelID, msgToSend)
		return
	}
	switch strings.ToLower(msg.Content) {
	case "settings":
		msgToSend := handleSettingMsg(msg.Author.Username, msg.Author.ID)
		session.ChannelMessageSendEmbed(msg.ChannelID, msgToSend)
		break
	case "help":
		session.ChannelMessageSendEmbed(msg.ChannelID, createCommandMenu())
		break
	case "graph":
		optionsToSend := []string{
			"bar",
			"pie",
		}
		userSettingsMap[msg.Author.ID] = &keepTrackOfMsg{
			command:   "graph",
			options:   optionsToSend,
			id:        msg.Author.ID,
			channelID: msg.ChannelID,
			session:   session,
		}
		userSettingsMap[msg.Author.ID].sendSettingMsg()
		break
	case "hide":
		optionsToSend := make([]string, 0)
		var results []stat
		db.findAll("gamestats", bson.M{"id": msg.Author.ID, "ignore": false}, &results)
		for _, item := range results {
			optionsToSend = append(optionsToSend, item.Game)
		}
		userSettingsMap[msg.Author.ID] = &keepTrackOfMsg{
			command:   "hide",
			options:   optionsToSend,
			id:        msg.Author.ID,
			channelID: msg.ChannelID,
			session:   session,
		}
		userSettingsMap[msg.Author.ID].sendSettingMsg()
		break
	case "show":
		optionsToSend := make([]string, 0)
		var results []stat
		db.findAll("gamestats", bson.M{"id": msg.Author.ID, "ignore": true}, &results)
		for _, item := range results {
			optionsToSend = append(optionsToSend, item.Game)
		}
		userSettingsMap[msg.Author.ID] = &keepTrackOfMsg{
			command:   "show",
			options:   optionsToSend,
			id:        msg.Author.ID,
			channelID: msg.ChannelID,
			session:   session,
		}
		userSettingsMap[msg.Author.ID].sendSettingMsg()
		break
	case "mention":
		optionsToSend := []string{
			"true",
			"false",
		}
		userSettingsMap[msg.Author.ID] = &keepTrackOfMsg{
			command:   "mention",
			options:   optionsToSend,
			id:        msg.Author.ID,
			channelID: msg.ChannelID,
			session:   session,
		}
		userSettingsMap[msg.Author.ID].sendSettingMsg()
		break
	default:
		session.ChannelMessageSend(msg.ChannelID, "Please Enter A Valid Setting.")
		break
	}
}

func handleGuildImgCreation(guildID, channelID string, session *discordgo.Session) (*discordgo.MessageSend, error) {
	guild, _ := session.State.Guild(guildID)
	var guildAvatar *image.Image
	if guild.Icon == "" {
		createdIcon, err := createGuildAvatar(guild.Name)
		if err != nil {
			return nil, errors.Wrap(err, "Creating Guild Avatar")
		}
		guildAvatar = createdIcon
	} else {
		guildIconGet, err := session.GuildIcon(guildID)
		if err != nil {
			return nil, errors.Wrap(err, "Getting Guild Icon")
		}
		guildAvatar = &guildIconGet
	}
	messageObj, err := processUserImg(guildID, guild.Name, guildAvatar)
	if err != nil {
		return nil, errors.Wrap(err, "Processing User IMG")
	}
	return messageObj, nil
}
