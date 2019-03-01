package main

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/globalsign/mgo/bson"
)

func createFakePresences(usersToLink []*discordgo.User, gameNameStr string) []*discordgo.Presence {
	presenceArr := make([]*discordgo.Presence, 0)
	for i := range usersToLink {
		if gameNameStr == "" {
			presenceArr = append(presenceArr, &discordgo.Presence{
				User: usersToLink[i],
				Game: nil,
			})
			continue
		}
		presenceArr = append(presenceArr, &discordgo.Presence{
			User: usersToLink[i],
			Game: &discordgo.Game{
				Name: gameNameStr,
			},
		})
	}
	return presenceArr
}

func createFakeGuild(presences []*discordgo.Presence, users []*discordgo.User) *discordgo.Guild {
	randID := rand.Intn(1000000)
	guildMembers := make([]*discordgo.Member, 0)
	for i := range users {
		guildMembers = append(guildMembers, &discordgo.Member{
			User: users[i],
		})
	}
	return &discordgo.Guild{
		ID:        strconv.Itoa(randID),
		Members:   guildMembers,
		Presences: presences,
	}
}

func createFakeMembers(isBot bool, howMany int) []*discordgo.User {
	memberArr := make([]*discordgo.User, 0)
	for i := 0; i < howMany; i++ {
		randID := rand.Intn(1000000)
		memberArr = append(memberArr, &discordgo.User{
			ID:  strconv.Itoa(randID),
			Bot: isBot,
		})
	}
	return memberArr
}

func cleanDiscordUsers(t *testing.T) {
	for _, item := range discordUsers {
		delete(discordUsers, item.userID)
	}
	if len(discordUsers) != 0 {
		t.Error("Clearing Failed")
	}
}

func TestSetUpDB(t *testing.T) {
	if db == nil {
		_, testDB := setUpDB("localhost/testing")
		db = testDB
	}
	db.removeAll("settings", bson.M{})
	db.removeAll("gamestats", bson.M{})
	db.removeAll("gameicons", bson.M{})
	db.removeAll("iconblacklists", bson.M{})
}

func TestAddGuild(t *testing.T) {
	TestSetUpDB(t)
	pMembers := createFakeMembers(false, 3)
	nMembers := createFakeMembers(false, 3)
	pPresences := createFakePresences(pMembers, "Test")
	nPresences := createFakePresences(nMembers, "")
	members := append(pMembers, nMembers...)
	presences := append(pPresences, nPresences...)
	guild := createFakeGuild(presences, members)

	newMembers := createFakeMembers(false, 3)
	newPresences := createFakePresences(newMembers, "Test2")
	newMembers = append(newMembers, pMembers...)
	newPresences = append(newPresences, pPresences...)
	guild2 := createFakeGuild(newPresences, newMembers)

	addDiscordGuild(guild)

	if len(discordUsers) == 0 {
		t.Error("Length is Equal to 0")
	}

	for i := range members {
		if presences[i].Game != nil {
			if discordUsers[members[i].ID].currentGame != presences[i].Game.Name {
				t.Error("Game name is not equal")
				t.Error("Expected: " + presences[i].Game.Name + "Got: " + discordUsers[members[i].ID].currentGame)
			}
		} else if discordUsers[members[i].ID].currentGame != "" {
			t.Error("Game is not equal")
			t.Error("Expected: " + "Got: " + discordUsers[members[i].ID].currentGame)
		}
	}

	addDiscordGuild(guild2)

	if len(discordUsers) != (len(members) + 3) {
		t.Error("Error adding guild with some same users")
	}

	cleanDiscordUsers(t)
}

func TestAddSingleUser(t *testing.T) {
	TestSetUpDB(t)
	members := createFakeMembers(false, 3)
	presences := createFakePresences(members, "Test2")
	guild := createFakeGuild(presences, members)
	addDiscordGuild(guild)

	memberToAdd := createFakeMembers(false, 1)
	addDiscordUser(memberToAdd[0].ID, guild.ID, false)

	if len(discordUsers) != 4 {
		t.Error("Discord Users Length Incorrent")
		t.Error(len(discordUsers))
	}

	addDiscordUser(memberToAdd[0].ID, guild.ID, false)

	if len(discordUsers) != 4 {
		t.Error("Discord Users Length Incorrent")
		t.Error(len(discordUsers))
	}

	cleanDiscordUsers(t)
}

func TestAddBlacklistGuild(t *testing.T) {
	TestSetUpDB(t)
	members := createFakeMembers(false, 1)
	presences := createFakePresences(members, "Test2")
	guild := createFakeGuild(presences, members)

	guildBlacklists = append(guildBlacklists, guild.ID)

	addDiscordGuild(guild)
	if len(discordUsers) != 0 {
		t.Error("Discord Users is Greater Than 0")
		t.Error(len(discordUsers))
	}
}

func TestAddBlacklistUser(t *testing.T) {
	TestSetUpDB(t)
	members := createFakeMembers(false, 1)
	presences := createFakePresences(members, "Test2")
	guild := createFakeGuild(presences, members)

	guildBlacklists = append(guildBlacklists, guild.ID)

	addDiscordUser(members[0].ID, guild.ID, false)
	if len(discordUsers) != 0 {
		t.Error("Discord Users is Greater Than 0")
		t.Error(len(discordUsers))
	}
}

func TestRemoveUserOneGuild(t *testing.T) {
	TestSetUpDB(t)
	memberP := createFakeMembers(false, 1)
	member := createFakeMembers(false, 1)
	presenceP := createFakePresences(memberP, "Test")
	presence := createFakePresences(member, "")
	members := append(memberP, member...)
	presences := append(presenceP, presence...)
	guild := createFakeGuild(presences, members)

	addDiscordGuild(guild)

	removeDiscordUser(memberP[0].ID, guild.ID)
	if len(discordUsers) != 1 {
		t.Error("Users is not equal to 1")
		t.Error("Got: " + strconv.Itoa(len(discordUsers)))
	}
	if db.itemExists("gamestats", bson.M{"id": memberP[0].ID}) == false {
		t.Error("Stat not saved")
	}
	if db.itemExists("gamestats", bson.M{"id": guild.ID}) == false {
		t.Error("Stat not saved for Guild")
	}

	removeDiscordUser(member[0].ID, guild.ID)
	if len(discordUsers) != 0 {
		t.Error("Users is not equal to 0")
		t.Error("Got: " + strconv.Itoa(len(discordUsers)))
	}
	if db.itemExists("gamestats", bson.M{"id": member[0].ID}) == true {
		t.Error("Stat save incorrect")
	}
	cleanDiscordUsers(t)
}

func TestRemoveUserDiffirentGuild(t *testing.T) {
	TestSetUpDB(t)
	memberP := createFakeMembers(false, 1)
	member := createFakeMembers(false, 1)
	presenceP := createFakePresences(memberP, "Test")
	presence := createFakePresences(member, "")
	members := append(memberP, member...)
	presences := append(presenceP, presence...)
	guild := createFakeGuild(presences, members)
	guild2 := createFakeGuild(presences, members)

	addDiscordGuild(guild)
	addDiscordGuild(guild2)

	if len(discordUsers[member[0].ID].otherGuilds) != 1 {
		t.Error("Other Guilds is not 1")
		t.Error("Got: " + strconv.Itoa(len(discordUsers[member[0].ID].otherGuilds)))
	}
	if len(discordUsers[memberP[0].ID].otherGuilds) != 1 {
		t.Error("Other Guilds is not 1")
		t.Error("Got: " + strconv.Itoa(len(discordUsers[memberP[0].ID].otherGuilds)))
	}

	removeDiscordUser(memberP[0].ID, guild.ID)
	if len(discordUsers) != 2 {
		t.Error("Users is not equal to 2")
		t.Error("Got: " + strconv.Itoa(len(discordUsers)))
	}
	if discordUsers[memberP[0].ID].isPlaying == false {
		t.Error("Is playing is false")
	}
	if db.itemExists("gamestats", bson.M{"id": guild.ID}) == false {
		t.Error("Stat not saved for Guild")
	}

	removeDiscordUser(member[0].ID, guild.ID)
	if len(discordUsers) != 2 {
		t.Error("Users is not equal to 2")
		t.Error("Got: " + strconv.Itoa(len(discordUsers)))
	}
	if db.itemExists("gamestats", bson.M{"id": member[0].ID}) == true {
		t.Error("Stat save incorrect")
	}
	cleanDiscordUsers(t)
}

func TestLoadDiscordAvatar(t *testing.T) {
	_, err := loadDiscordAvatar("https://nerdfox.me")
	if err == nil {
		t.Error("No Error When Loading Fake Img")
	}
	_, err = loadDiscordAvatar("https://nerdfox.me/static/img/art/2BOxJGumR.png")
	if err != nil {
		t.Error("Got Error")
		t.Error(err)
	}
}

func TestGetGameImg(t *testing.T) {
	TestSetUpDB(t)
	keys, _ := ioutil.ReadFile("private.txt")
	keysSplit := strings.Split(string(keys), "\n")
	apiKey = keysSplit[1]
	cseID = keysSplit[0]
	dataDir = "/Users/dasfox/Desktop/Go/data/stats"
	gameImgDir = dataDir + "/Images/Game"

	err := getGameImg("Spotify")
	if err != nil {
		t.Error("Got Error Spotify")
		t.Error(err)
	}
	if db.itemExists("gameicons", bson.M{"game": "Spotify"}) == false {
		t.Error("Item does not exsist in DB")
		t.FailNow()
	}
	var gameInfo icon
	db.findOne("gameicons", bson.M{"game": "Spotify"}, &gameInfo)
	if _, err := os.Stat(path.Join(dataDir, gameInfo.Location)); os.IsNotExist(err) {
		t.Error("File Does Not Exsist")
	}
	//os.Remove(path.Join(dataDir, gameInfo.Location))

	err = getGameImg("odgugofidugfdoigiofdgfd7g98fdg89df7g98df7gfdg")
	if err != nil {
		t.Error("Got No Error Random")
	}
	if db.itemExists("iconblacklists", bson.M{"game": "odgugofidugfdoigiofdgfd7g98fdg89df7g98df7gfdg"}) == false {
		t.Error("Item does not exsist in DB blacklist")
	}
}

func TestPresenceUpdateGame(t *testing.T) {
	TestSetUpDB(t)
	members := createFakeMembers(false, 3)
	presences := createFakePresences(members, "Test")
	guild1Members := make([]*discordgo.User, 0)
	guild2Members := make([]*discordgo.User, 0)
	guild1Presences := make([]*discordgo.Presence, 0)
	guild2Presences := make([]*discordgo.Presence, 0)
	guild1Members = append(guild1Members, members[0])
	guild1Members = append(guild1Members, members[1])
	guild2Members = append(guild2Members, members[0])
	guild2Members = append(guild2Members, members[2])
	guild1Presences = append(guild1Presences, presences[0])
	guild1Presences = append(guild1Presences, presences[1])
	guild2Presences = append(guild2Presences, presences[0])
	guild2Presences = append(guild2Presences, presences[2])
	guild1 := createFakeGuild(guild1Presences, guild1Members)
	guild2 := createFakeGuild(guild2Presences, guild2Members)
	addDiscordGuild(guild1)
	addDiscordGuild(guild2)

	if len(discordUsers) != 3 {
		t.Error("Discors users is not 3")
		t.Error(len(discordUsers))
	}

	for _, user := range discordUsers {
		presenceUpdate := &discordgo.PresenceUpdate{
			Presence: discordgo.Presence{
				Game: nil,
				User: &discordgo.User{
					ID: user.userID,
				},
			},
		}
		handlePresenceUpdate(presenceUpdate)
		if db.itemExists("gamestats", bson.M{"id": user.userID, "game": "Test"}) == false {
			t.Error("Game stat not saved")
		}
		if db.itemExists("gamestats", bson.M{"id": user.mainGuild, "game": "Test"}) == false {
			t.Error("Game Guild stat not saved")
		}
		for key := range user.otherGuilds {
			if db.itemExists("gamestats", bson.M{"id": key, "game": "Test"}) == false {
				t.Error("Game other Guild stat not saved")
			}
		}
	}
	cleanDiscordUsers(t)
}

func TestPresenceUpdateNoGame(t *testing.T) {
	TestSetUpDB(t)
	members := createFakeMembers(false, 3)
	presences := createFakePresences(members, "")
	guild1Members := make([]*discordgo.User, 0)
	guild2Members := make([]*discordgo.User, 0)
	guild1Presences := make([]*discordgo.Presence, 0)
	guild2Presences := make([]*discordgo.Presence, 0)
	guild1Members = append(guild1Members, members[0])
	guild1Members = append(guild1Members, members[1])
	guild2Members = append(guild2Members, members[0])
	guild2Members = append(guild2Members, members[2])
	guild1Presences = append(guild1Presences, presences[0])
	guild1Presences = append(guild1Presences, presences[1])
	guild2Presences = append(guild2Presences, presences[0])
	guild2Presences = append(guild2Presences, presences[2])
	guild1 := createFakeGuild(guild1Presences, guild1Members)
	guild2 := createFakeGuild(guild2Presences, guild2Members)
	addDiscordGuild(guild1)
	addDiscordGuild(guild2)

	if len(discordUsers) != 3 {
		t.Error("Discors users is not 3")
		t.Error(len(discordUsers))
	}

	for _, user := range discordUsers {
		presenceUpdate := &discordgo.PresenceUpdate{
			Presence: discordgo.Presence{
				Game: nil,
				User: &discordgo.User{
					ID: user.userID,
				},
			},
		}
		handlePresenceUpdate(presenceUpdate)
		if db.itemExists("gamestats", bson.M{"id": user.userID, "game": "Test"}) == true {
			t.Error("Game stat saved")
		}
		if db.itemExists("gamestats", bson.M{"id": user.mainGuild, "game": "Test"}) == true {
			t.Error("Game Guild stat saved")
		}
	}
	cleanDiscordUsers(t)
}

func TestPresenceUpdateNoGamePlaying(t *testing.T) {
	TestSetUpDB(t)
	members := createFakeMembers(false, 2)
	presences := createFakePresences(members, "")
	guild := createFakeGuild(presences, members)
	addDiscordGuild(guild)

	if len(discordUsers) != 2 {
		t.Error("Discors users is not 2")
		t.Error(len(discordUsers))
	}

	for _, user := range discordUsers {
		presenceUpdate := &discordgo.PresenceUpdate{
			Presence: discordgo.Presence{
				Game: &discordgo.Game{
					Name: "Test",
				},
				User: &discordgo.User{
					ID: user.userID,
				},
			},
		}
		handlePresenceUpdate(presenceUpdate)
		if user.currentGame != "Test" {
			t.Error("User Game is Not Test")
			t.Error(user.currentGame)
		}
		if user.isPlaying == false {
			t.Error("Is playing is not true")
		}
	}
	cleanDiscordUsers(t)
}

func TestPresenceUpdateChangeGamePlaying(t *testing.T) {
	TestSetUpDB(t)
	members := createFakeMembers(false, 2)
	presences := createFakePresences(members, "Test")
	guild := createFakeGuild(presences, members)
	addDiscordGuild(guild)

	if len(discordUsers) != 2 {
		t.Error("Discors users is not 2")
		t.Error(len(discordUsers))
	}

	for _, user := range discordUsers {
		oldTime := user.startedPlaying
		presenceUpdate := &discordgo.PresenceUpdate{
			Presence: discordgo.Presence{
				Game: &discordgo.Game{
					Name: "Test2",
				},
				User: &discordgo.User{
					ID: user.userID,
				},
			},
		}
		handlePresenceUpdate(presenceUpdate)
		if db.itemExists("gamestats", bson.M{"id": user.userID, "game": "Test"}) == false {
			t.Error("Game stat not saved")
		}
		if db.itemExists("gamestats", bson.M{"id": user.mainGuild, "game": "Test"}) == false {
			t.Error("Game Guild stat not saved")
		}
		if user.isPlaying != true {
			t.Error("Is playing is not true")
		}
		if user.currentGame != "Test2" {
			t.Error("Game not Test2")
		}
		if oldTime.Unix() != user.startedPlaying.Unix() {
			t.Error("Time is the same")
			t.Error(oldTime)
			t.Error(user.startedPlaying)
		}
	}
	cleanDiscordUsers(t)
}
