package main

import (
	"math/rand"
	"strconv"
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

func TestSetUpDB(t *testing.T) {
	if db == nil {
		_, testDB := setUpDB("localhost/testing")
		db = testDB
	}
	db.removeAll("settings", bson.M{})
	db.removeAll("gamestats", bson.M{})
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

	for _, item := range discordUsers {
		delete(discordUsers, item.userID)
	}
	if len(discordUsers) != 0 {
		t.Error("Clearing Failed")
	}
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

	for _, item := range discordUsers {
		delete(discordUsers, item.userID)
	}
	if len(discordUsers) != 0 {
		t.Error("Clearing Failed")
	}
}

func TestRemoveUser(t *testing.T) {

}
