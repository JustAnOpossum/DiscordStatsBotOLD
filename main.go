package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/pkg/errors"
)

var db *datastore
var out = ioutil.Discard

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
			} else if ok := discordUsers[userID].otherGuilds[guildID]; ok == nil {
				fmt.Println(discordUsers[userID].otherGuilds[guildID] == nil)
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

func removeDiscordUser(session *discordgo.Session, userID string) {
	user := discordUsers[userID]
	startingGuildID := user.mainGuildID
	for guildID, guild := range user.otherGuilds {
		for _, member := range guild.Members {
			if member.User.ID == userID {
				user.mainGuildID = guild.ID
				delete(user.otherGuilds, guildID)
				break
			} else {
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
	fmt.Println(discordUsers["68553027849564160"])
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

func presenceUpdate(session *discordgo.Session, presence *discordgo.PresenceUpdate) {
	game := presence.Game
	user := discordUsers[presence.User.ID]
	if user.mainGuildID == presence.GuildID {
		if game != nil { //Started Playing Game
			if game.Name != user.currentGame {
				fmt.Fprintln(out, "Started Playing Game "+game.Name)
				if user.isPlaying == true { //Switching from other game
					fmt.Fprintln(out, "Switching From Other Game "+user.currentGame)
					user.save()
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
				user.reset()
			}

		}
	}
}

func loadDiscordAvatar(url string) (image.Image, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	res, err := httpClient.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "Making Discord HTTP Avatar Request")
	}
	decodedImg, _, err := image.Decode(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Deconding Image")
	}
	return decodedImg, nil
}
