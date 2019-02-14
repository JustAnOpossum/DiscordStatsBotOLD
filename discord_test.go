package main

import (
	"io/ioutil"
	"math/rand"
	"strconv"
	"testing"
	"time"

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

	time.Sleep(time.Second * 5)

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

	time.Sleep(time.Second * 5)

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

func TestGetImage(t *testing.T) {
	keys := ioutil.ReadFile("private.txt")
}
