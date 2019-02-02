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
		guildMemebers, err := session.Guild(guild.ID)
		if err != nil {
			panic(err)
		}
		for _, presence := range guildMemebers.Presences {
			userID := presence.User.ID
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
				currentGame:    currentGame,
				startedPlaying: startedPlaying,
				isPlaying:      isPlaying,
			}
		}
	}
}

func presenceUpdate(session *discordgo.Session, presence *discordgo.PresenceUpdate) {
	game := presence.Game
	user := discordUsers[presence.User.ID]
	if game != nil { //Started Playing Game
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
	} else { //Stopped Playing Game
		fmt.Fprintln(out, "Stopped Playing Game")
		user.save()
		user.reset()
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
