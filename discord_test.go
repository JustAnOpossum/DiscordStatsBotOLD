package main

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/bwmarrin/discordgo"
)

type testUser struct {
	ID      string
	guildID string
	bot     bool
}

var testUsers = []testUser{
	testUser{ //User1
		ID:      "123",
		guildID: "1234",
		bot:     false,
	},
	testUser{ //User2
		ID:      "456",
		guildID: "1234",
		bot:     false,
	},
	testUser{ //User1 Diffirent Guild
		ID:      "123",
		guildID: "4567",
		bot:     false,
	},
}

var testGuilds = []*discordgo.Guild{
	&discordgo.Guild{
		ID: "1234",
		Presences: []*discordgo.Presence{
			&discordgo.Presence{
				User: &discordgo.User{
					ID: testUsers[0].ID,
				},
				Game: &discordgo.Game{
					Name: "Test",
				},
			},
			&discordgo.Presence{
				User: &discordgo.User{
					ID: testUsers[1].ID,
				},
				Game: nil,
			},
		},
		Members: []*discordgo.Member{
			&discordgo.Member{
				User: &discordgo.User{
					ID: testUsers[0].ID,
				},
			},
			&discordgo.Member{
				User: &discordgo.User{
					ID: testUsers[1].ID,
				},
			},
		},
	},
	&discordgo.Guild{
		ID: "5678",
		Presences: []*discordgo.Presence{
			&discordgo.Presence{
				User: &discordgo.User{
					ID: testUsers[0].ID,
				},
				Game: &discordgo.Game{
					Name: "Test",
				},
			},
			&discordgo.Presence{
				User: &discordgo.User{
					ID: testUsers[1].ID,
				},
				Game: nil,
			},
		},
		Members: []*discordgo.Member{
			&discordgo.Member{
				User: &discordgo.User{
					ID: testUsers[0].ID,
				},
			},
			&discordgo.Member{
				User: &discordgo.User{
					ID: testUsers[1].ID,
				},
			},
		},
	},
}

func createFakePresence(userToLink *discordgo.User, gameNameStr string) *discordgo.Presence {
	if gameNameStr == "" {
		return &discordgo.Presence{
			User: userToLink,
			Game: nil,
		}
	}
	return &discordgo.Presence{
		User: userToLink,
		Game: &discordgo.Game{
			Name: gameNameStr,
		},
	}
}

func createFakeGuild(presences []*discordgo.Presence) *discordgo.Guild {
	randID := rand.Intn(1000000)
	return &discordgo.Guild{
		ID: strconv.Itoa(randID),
	}
}

func createFakeMember(isBot bool) *discordgo.Member {
	randID := rand.Intn(1000000)
	return &discordgo.Member{
		User: &discordgo.User{
			ID: strconv.Itoa(randID),
		},
	}
}

func TestCreate(t *testing.T) {
	m1 := createFakeMember(false)
	g1 := createFakeGuild()
	p1 := createFakePresence(m1)
}

func TestSetUpDB(t *testing.T) {
	if db == nil {
		_, testDB := setUpDB("localhost/testing")
		db = testDB
	}
}

func TestAddUser(t *testing.T) {
	TestSetUpDB(t)
	testUser := testUsers[0]
	addDiscordUser(testUser.ID, testUser.guildID, testUser.bot)

	if discordUsers[testUser.ID].userID != testUser.ID {
		t.Error("User ID is not equal to Test ID")
	}
	if discordUsers[testUser.ID].mainGuild != testUser.guildID {
		t.Error("Guild ID is not equal to Test ID")
	}
	delete(discordUsers, testUser.ID)
}

func TestAddSameUser(t *testing.T) {
	TestSetUpDB(t)
	testUser := testUsers[0]
	testUserSame := testUsers[0]
	addDiscordUser(testUser.ID, testUser.guildID, testUser.bot)
	addDiscordUser(testUserSame.ID, testUserSame.guildID, testUserSame.bot)

	if len(discordUsers) > 1 {
		t.Error("Length is GT 1")
		t.FailNow()
	}
	delete(discordUsers, testUser.ID)
}

func TestAddUserDiffirentGuild(t *testing.T) {
	TestSetUpDB(t)
	testUser := testUsers[0]
	testUserGuild := testUsers[2]
	addDiscordUser(testUser.ID, testUser.guildID, testUser.bot)
	addDiscordUser(testUserGuild.ID, testUserGuild.guildID, testUserGuild.bot)

	if len(discordUsers[testUser.ID].otherGuilds) == 0 {
		t.Error("otherGuild length is equal to 0")
	}
	delete(discordUsers, testUser.ID)
	delete(discordUsers, testUserGuild.ID)
}

func TestAddGuild(t *testing.T) {
	TestSetUpDB(t)
	addDiscordGuild(testGuilds[0])

	if len(discordUsers) == 0 {
		t.Error("Length of users is 0 ")
		t.Error(discordUsers)
	}
	if discordUsers[testUsers[0].ID].isPlaying == false {
		t.Error("Is playing is false User1")
	}
	if discordUsers[testUsers[0].ID].currentGame == "" {
		t.Error("Game name is wrong User1")
	}

	if discordUsers[testUsers[1].ID].currentGame != "" {
		t.Error("Game name is wrong User2")
		t.Error(discordUsers[testUsers[2].ID].currentGame)
	}
	delete(discordUsers, testUsers[0].ID)
	delete(discordUsers, testUsers[1].ID)
}

func TestAddGuildSameUsers(t *testing.T) {
	TestSetUpDB(t)
	addDiscordGuild(testGuilds[0])
	addDiscordGuild(testGuilds[1])

	if len(discordUsers) == len(testGuilds[0].Members) {
		t.Error("Length of users is 0 ")
		t.Error(discordUsers)
	}
	if discordUsers[testUsers[0].ID].isPlaying == false {
		t.Error("Is playing is false User1")
	}
	if discordUsers[testUsers[0].ID].currentGame == "" {
		t.Error("Game name is wrong User1")
	}

	if discordUsers[testUsers[1].ID].currentGame != "" {
		t.Error("Game name is wrong User2")
		t.Error(discordUsers[testUsers[2].ID].currentGame)
	}
	delete(discordUsers, testUsers[0].ID)
	delete(discordUsers, testUsers[1].ID)
}
