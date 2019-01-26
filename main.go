package main

import (
	"fmt"
	"image"
	"net/http"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

var db *datastore

func main() {
	// img, _ := loadDiscordAvatar("https://nerdfox.me/static/img/art/Zs9y9z44-.jpg")
	// createImage(img, "20", "30", "DasFox", barChart)
	session, dbStruct := setUpDB()
	db = dbStruct
	defer session.Close()

	// barChart, _ := createChart("bar", "68553027849564160")
	// ioutil.WriteFile("test.png", barChart.Bytes(), 0666)
	var test []gameStatsStruct
	db.findAll("gameicons", bson.M{}, &test)
	fmt.Println(test)
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
