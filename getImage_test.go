package main

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"
)

func TestDownloadImg(t *testing.T) {
	_, err := downloadImg("https://nerdfox.me", "image/png")
	if err == nil {
		t.Error("No Error When Loading Fake Img")
	}
	_, err = downloadImg("https://nerdfox.me/static/img/art/2BOxJGumR.png", "image/jpeg")
	if err == nil {
		t.Error("No Error When Forcing PNG as JPEG")
	}

	_, err = downloadImg("https://nerdfox.me/static/img/art/2BOxJGumR.png", "image/png")
	if err != nil {
		t.Error("Error when Correct Img Passed")
		t.Error(err)
	}
}

func TestGetImageFromBing(t *testing.T) {
	keys, _ := ioutil.ReadFile("private.txt")
	bingToken = string(keys)

	searchResults, err := getImagesFromBing("Spotify")
	if err != nil {
		t.Error("Error Getting Bing Images")
		t.Error(err)
		t.FailNow()
	}
	if len(searchResults.Value) == 0 {
		t.Error("Search Results Equal to 0")
		t.Error(searchResults.Value)
	}

	time.Sleep(time.Second * 2)

	searchResults, err = getImagesFromBing("78fd9g7fdvhudrhui4578546rehkujthkejrhtrt")
	if err != nil {
		t.Error("Error Getting Bing Images")
		t.Error(err)
		t.FailNow()
	}
	if len(searchResults.Value) != 0 {
		t.Error("Search Results Not Equal to 0")
	}
}

func TestProcessImg(t *testing.T) {
	TestSetUpDB(t)
	loadImg, err := downloadImg("https://nerdfox.me/static/img/art/2BOxJGumR.png", "image/png")
	if err != nil {
		t.Error("Error Loading IMG")
		t.FailNow()
	}
	dataDirFile, _ := ioutil.ReadFile("dataDir.txt")
	dataDir = string(dataDirFile)
	gameImgDir = dataDir + "/Images/Game"
	testImgItem := bingAnswer{
		Value: []bingValue{
			bingValue{
				ContentURL:     "https://nerdfox.me/static/img/art/2BOxJGumR.png",
				EncodingFormat: "png",
			},
		},
	}

	err = processImg(testImgItem.Value[0].EncodingFormat, loadImg, "test")
	if err != nil {
		t.Error("Error Processing Img")
		t.Error(err)
		t.FailNow()
	}
	var gameName icon
	db.findOne("gameicons", bson.M{"game": "test"}, &gameName)
	os.Remove(path.Join(dataDir, gameName.Location))
}

func TestTop5Normal(t *testing.T) {
	TestSetUpDB(t)
	dataDirFile, _ := ioutil.ReadFile("dataDir.txt")
	keyFile, _ := ioutil.ReadFile("private.txt")
	dataDir = string(dataDirFile)
	gameImgDir = dataDir + "/Images/Game"
	bingToken = string(keyFile)

	for i := 0; i < 5; i++ {
		itemToInsert := stat{
			ID:    "123",
			Game:  "Test" + strconv.Itoa(i),
			Hours: float64(i),
		}
		db.insert("gamestats", itemToInsert)
	}
	var added []stat
	db.findAll("gamestats", bson.M{}, &added)
	if len(added) != 5 {
		t.Error("Length is Not 5")
		t.Error(len(added))
	}
	getTop5Img("123")

	var gameIcons []icon
	db.findAll("gameicons", bson.M{}, &gameIcons)
	if len(gameIcons) != 5 {
		t.Error("Database is Not 5")
		t.Error(len(gameIcons))
	}

	for i := 0; i < 5; i++ {
		var gameName icon
		db.findOne("gameicons", bson.M{"game": "Test" + strconv.Itoa(i)}, &gameName)
		os.Remove(path.Join(dataDir, gameName.Location))
	}
}

func TestTop5NormalBlacklist(t *testing.T) {
	TestSetUpDB(t)
	dataDirFile, _ := ioutil.ReadFile("dataDir.txt")
	keyFile, _ := ioutil.ReadFile("private.txt")
	dataDir = string(dataDirFile)
	gameImgDir = dataDir + "/Images/Game"
	bingToken = string(keyFile)

	for i := 0; i < 7; i++ {
		var itemToInsert stat
		if i <= 4 {
			itemToInsert = stat{
				ID:    "123",
				Game:  "Test" + strconv.Itoa(i),
				Hours: float64(i),
			}
		} else {
			itemToInsert = stat{
				ID:    "123",
				Game:  "gdfjilhogjudfhiguj8h7uf8ghu8f9guhfh" + strconv.Itoa(i),
				Hours: float64(i),
			}
		}
		db.insert("gamestats", itemToInsert)
	}
	var added []stat
	db.findAll("gamestats", bson.M{}, &added)
	if len(added) != 7 {
		t.Error("Length is Not 7")
		t.Error(len(added))
	}
	getTop5Img("123")

	var gameIcons []icon
	db.findAll("gameicons", bson.M{}, &gameIcons)
	if len(gameIcons) != 5 {
		t.Error("Database is Not 5")
		t.Error(len(gameIcons))
	}

	for i := 0; i < 5; i++ {
		var gameName icon
		db.findOne("gameicons", bson.M{"game": "Test" + strconv.Itoa(i)}, &gameName)
		os.Remove(path.Join(dataDir, gameName.Location))
	}
}

func TestGetGameImg(t *testing.T) {
	TestSetUpDB(t)
	keys, _ := ioutil.ReadFile("private.txt")
	bingToken = string(keys)
	dataDirFile, _ := ioutil.ReadFile("dataDir.txt")
	dataDir = string(dataDirFile)
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
	os.Remove(path.Join(dataDir, gameInfo.Location))

	err = getGameImg("odgugofidugfdoigiofdgfd7g98fdg89df7g98df7gfdg")
	if err == nil {
		t.Error("Got No Error Random")
	}
	if db.itemExists("iconblacklists", bson.M{"game": "odgugofidugfdoigiofdgfd7g98fdg89df7g98df7gfdg"}) == false {
		t.Error("Item does not exsist in DB blacklist")
	}
}
