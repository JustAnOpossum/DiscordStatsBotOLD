package main

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
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

func TestGetImageFromGoogle(t *testing.T) {
	keys, _ := ioutil.ReadFile("private.txt")
	keysSplit := strings.Split(string(keys), "\n")
	apiKey = keysSplit[1]
	cseID = keysSplit[0]

	searchResults, err := getImagesFromGoogle("Spotify")
	if err != nil {
		t.Error("Error Getting Google Images")
		t.Error(err)
		t.FailNow()
	}
	if len(searchResults.Items) == 0 {
		t.Error("Search Results Equal to 0")
		t.Error(searchResults.Items)
	}

	time.Sleep(time.Second * 2)

	searchResults, err = getImagesFromGoogle("78fd9g7fdvhudrhui4578546rehkujthkejrhtrt")
	if err != nil {
		t.Error("Error Getting Google Images")
		t.Error(err)
		t.FailNow()
	}
	if len(searchResults.Items) != 0 {
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
	testImgItem := imgItem{
		Link: "https://nerdfox.me/static/img/art/2BOxJGumR.png",
		Mime: "image/png",
	}

	err = processImg(testImgItem, loadImg, "test")
	if err != nil {
		t.Error("Error Processing Img")
		t.Error(err)
		t.FailNow()
	}
	var gameName icon
	db.findOne("gameicons", bson.M{"game": "test"}, &gameName)
	os.Remove(path.Join(dataDir, gameName.Location))
}
