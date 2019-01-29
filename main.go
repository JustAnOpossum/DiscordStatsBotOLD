package main

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

var db *datastore

func main() {
	img, _ := loadDiscordAvatar("https://cdn.discordapp.com/avatars/461294052529143825/ccf81d3c7ee8cf6b794cfbd81f0c9889.png?size=512")
	session, dbStruct := setUpDB()
	db = dbStruct
	defer session.Close()
	// err := createImage(&img, "21", "30", "DasFox", "bar", "68553027849564160")
	// fmt.Println(err)
	createBotImage(&img, "Stats Bot", "900", "400", "1000", "5", "80")
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
