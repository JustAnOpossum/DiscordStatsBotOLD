package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

var db *datastore

func main() {
	img, _ := loadDiscordAvatar("https://nerdfox.me/static/img/art/Zs9y9z44-.jpg")
	session, dbStruct := setUpDB()
	db = dbStruct
	defer session.Close()
	barChart, err := createChart("bar", "68553027849564160", &img)
	if err != nil {
		fmt.Println(err)
	}
	err = createImage(&img, "20", "30", "DasFox", barChart)
	fmt.Println(err)
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
