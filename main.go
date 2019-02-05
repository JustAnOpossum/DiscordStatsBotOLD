package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
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

func removeDiscordUser(session *discordgo.Session, userID string) {
	user := discordUsers[userID]
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

func presenceUpdate(session *discordgo.Session, presence *discordgo.PresenceUpdate) {
	if _, ok := discordUsers[presence.User.ID]; ok == true {
		game := presence.Game
		user := discordUsers[presence.User.ID]
		if user.mainGuildID == presence.GuildID {
			// if db.itemExists("gameicons", bson.M{"game": game.Name}) == false && db.itemExists("iconblacklists", bson.M{"game": game.Name}) == false {
			// 	getGameImg(game.Name)
			// }
			// if db.itemExists("iconblacklists", bson.M{"game": game.Name}) == true {
			// 	return
			// }
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
	}
}
