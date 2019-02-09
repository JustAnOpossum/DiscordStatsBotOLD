package main

import (
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"regexp"
	"strconv"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/globalsign/mgo/bson"
)

var db *datastore
var out = ioutil.Discard

//const dataDir string = "/mnt/c/Users/camer/Desktop/GO/Data/stats"
const dataDir string = "/Users/dasfox/Desktop/Go/data/stats"
const gameImgDir string = dataDir + "/Images/Game"

func main() {
	if os.Getenv("DEBUG") == "true" {
		out = os.Stdout
	}

	session, dbStruct := setUpDB()
	db = dbStruct
	defer session.Close()

	discord, err := discordgo.New("Bot " + os.Getenv("TOKEN"))
	if err != nil {
		panic(err)
	}

	discord.AddHandler(onReady)
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

	fmt.Println("Bot is started")
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-exitChan

	discord.Close()
}

func onReady(session *discordgo.Session, ready *discordgo.Ready) {
	guilds := ready.Guilds
	for _, guild := range guilds {
		addDiscordGuild(session, guild.ID)
	}
}

func guildCreate(session *discordgo.Session, guild *discordgo.GuildCreate) {
	addDiscordGuild(session, guild.ID)
}

func guildDeleted(session *discordgo.Session, deletedGuild *discordgo.GuildDelete) {
	for _, user := range discordUsers {
		if user.mainGuildID == deletedGuild.ID {
			removeDiscordUser(session, user.userID)
		}
	}
}

func memberDeleted(session *discordgo.Session, deletedMember *discordgo.GuildMemberRemove) {
	if _, ok := discordUsers[deletedMember.User.ID]; ok == true {
		removeDiscordUser(session, deletedMember.User.ID)
	}
}

func memberAdded(session *discordgo.Session, addedMember *discordgo.GuildMemberAdd) {
	addDiscordUser(session, addedMember.User.ID, addedMember.GuildID, addedMember.User.Bot)
}

func newMessage(session *discordgo.Session, msg *discordgo.MessageCreate) {
	currentWaitMsg := &waitingMsg{
		middleMsg:      "Creating Your Image, Please Wait...",
		currentSession: session,
	}
	if msg.GuildID == "" { //Private message handaler
		handlePrivateMessage(session, msg)
	}
	botUser, _ := session.User("@me")
	if msg.Author.ID == botUser.ID || msg.Author.Bot == true { //Make sure bot message don't repeat
		return
	}
	mentions := msg.Mentions
	switch len(mentions) {
	case 1:
		if mentions[0].ID == botUser.ID { //If only the bot is mentioned
			isGraphTypeGuild, _ := regexp.MatchString("guild", msg.Content)
			if isGraphTypeGuild == true { //Guild Handaler
				currentWaitMsg.send(msg.ChannelID)
				guild, _ := session.State.Guild(msg.GuildID)
				var guildAvatar *image.Image
				if guild.Icon == "" {
					createdIcon, err := createGuildAvatar(guild.Name)
					if err != nil {
						handleErrorInCommand(session, msg.ChannelID, err, currentWaitMsg)
						return
					}
					guildAvatar = createdIcon
				} else {
					guildIconGet, err := session.GuildIcon(msg.GuildID)
					if err != nil {
						handleErrorInCommand(session, msg.ChannelID, err, currentWaitMsg)
						return
					}
					guildAvatar = &guildIconGet
				}
				messageObj, err := processUserImg(msg.GuildID, guild.Name, guildAvatar)
				if err != nil {
					handleErrorInCommand(session, msg.ChannelID, err, currentWaitMsg)
					return
				}
				session.ChannelMessageSendComplex(msg.ChannelID, messageObj)
				break

			} //Normal user handeler
			isHelpMsg, _ := regexp.MatchString("help", msg.Content) //If help message
			if isHelpMsg == true {
				userDM, _ := session.UserChannelCreate(msg.Author.ID)
				var ignoredStats []stat
				var userSettings setting
				var unignoredStats []stat
				db.findAll("gamestats", bson.M{"id": msg.Author.ID, "ignore": true}, &ignoredStats)
				db.findOne("settings", bson.M{"id": msg.Author.ID}, &userSettings)
				db.findAll("gamestats", bson.M{"id": msg.Author.ID, "ignore": false}, &unignoredStats)
				session.ChannelMessageSendEmbed(userDM.ID, createMainMenu(strconv.Itoa(len(ignoredStats)), strconv.Itoa(len(unignoredStats)), userSettings.GraphType, userSettings.MentionForStats, msg.Author.Username))
				return
			}
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
			messageObj, err := processUserImg(msg.Author.ID, msg.Author.Username, userAvatar)
			if err != nil {
				handleErrorInCommand(session, msg.ChannelID, err, currentWaitMsg)
				return
			}
			session.ChannelMessageSendComplex(msg.ChannelID, messageObj)
			break
		}
		break
	case 2:
		if mentions[1].ID != botUser.ID {
			return
		}
		if mentions[0].Bot == true {

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
	}
	if currentWaitMsg.msgID != "" {
		currentWaitMsg.delete()
		currentMsgCountFile, err := ioutil.ReadFile(path.Join(dataDir, "botImg.txt"))
		if err != nil {
			return
		}
		currentMsgCount, _ := strconv.Atoi(string(currentMsgCountFile))
		currentMsgCount++
		err = ioutil.WriteFile(path.Join(dataDir, "botImg.txt"), []byte(strconv.Itoa(currentMsgCount)), 0644)
		if err != nil {
			return
		}
	}
}

func handlePrivateMessage(session *discordgo.Session, msg *discordgo.MessageCreate) {
	switch msg.Content {
	case "help":
		var ignoredStats []stat
		var unignoredStats []stat
		var userSettings setting
		db.findAll("gamestats", bson.M{"id": msg.Author.ID, "ignore": true}, &ignoredStats)
		db.findOne("settings", bson.M{"id": msg.Author.ID}, &userSettings)
		db.findAll("gamestats", bson.M{"id": msg.Author.ID, "ignore": false}, &unignoredStats)
		session.ChannelMessageSendEmbed(msg.ChannelID, createMainMenu(strconv.Itoa(len(ignoredStats)), strconv.Itoa(len(unignoredStats)), userSettings.GraphType, userSettings.MentionForStats, msg.Author.Username))
		break
	case "graph":

		break
	case "hide":
		break
	case "show":
		break
	case "mention":
		break
	case "show ignore":
		break
	default:
		break
	}
}

func presenceUpdate(session *discordgo.Session, presence *discordgo.PresenceUpdate) {
	if _, ok := discordUsers[presence.User.ID]; ok == true {
		game := presence.Game
		user := discordUsers[presence.User.ID]
		if user.mainGuildID == presence.GuildID {
			user.mu.Lock()
			defer user.mu.Unlock()
			if game != nil { //Started Playing Game
				if db.itemExists("gameicons", bson.M{"game": game.Name}) == false && db.itemExists("iconblacklists", bson.M{"game": game.Name}) == false {
					getGameImg(game.Name)
				}
				if db.itemExists("iconblacklists", bson.M{"game": game.Name}) == true {
					if db.itemExists("gamestats", bson.M{"game": game.Name}) == true {
						db.removeAll("gamestats", bson.M{"game": game.Name})
					}
					return
				}
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
	}
}
