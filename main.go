package main

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

var db *datastore

func main() {
	// img, _ := loadDiscordAvatar("https://cdn.discordapp.com/avatars/208075632746168322/8d4de5cdf920194f77cb2931c84cbc4a.png?size=512")
	session, dbStruct := setUpDB()
	db = dbStruct
	defer session.Close()
	// err := createImage(&img, "21", "30", "DasFox", "pie", "208075632746168322")
	// fmt.Println(err)
	db.countHours("gamestats", bson.M{"id": "208075632746168322"})
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
