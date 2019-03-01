package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/teris-io/shortid"
)

type googleJSON struct {
	Items []imgItem `json:"items"`
}

type imgItem struct {
	Link string `json:"link"`
	Mime string `json:"mime"`
}

var apiKey = os.Getenv("CSETOKEN")
var cseID = os.Getenv("CSEID")

func processImg(img imgItem, imgBuffer *bytes.Buffer, gameName string) error {
	fmt.Fprintln(out, "Got good image")
	shortID, _ := shortid.Generate()
	var ext string
	if img.Mime == "image/jpeg" {
		ext = ".jpg"
	}
	if img.Mime == "image/png" {
		ext = ".png"
	}
	var fileName = shortID + ext
	err := ioutil.WriteFile(path.Join(gameImgDir, fileName), imgBuffer.Bytes(), 0644)
	if err != nil {
		return errors.Wrap(err, "Writing Img")
	}
	fmt.Fprintln(out, "Wrote File")
	imgDecode, _, err := image.Decode(imgBuffer)
	fmt.Fprintln(out, "Decoded Img")
	if err != nil {
		fmt.Println(err)
		return errors.Wrap(err, "Decoding Img")
	}
	imgColors, err := getColorPallete(&imgDecode)
	if err != nil {
		fmt.Println(err)
		return errors.Wrap(err, "Getting Img colors")
	}
	fmt.Fprintln(out, "Got colors")
	itemToInsert := icon{
		Game:     gameName,
		Location: "Images/Game/" + fileName,
		Color:    imgColors.Main.RGBHex(),
	}
	db.insert("gameicons", itemToInsert)
	fmt.Fprintln(out, "Inserted IMG")
	return nil
}

func getGameImg(gameName string) error {
	imgArr, err := getImagesFromGoogle(gameName)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Error Getting Google Images"))
	}
	for _, img := range imgArr.Items {
		if img.Mime == "image/png" {
			imgBuffer, err := downloadImg(img.Link, "image/png")
			if err == nil {
				err = processImg(img, imgBuffer, gameName)
				if err != nil {
					return errors.Wrap(err, "Processing IMG")
				}
				return nil
			}
		}
	}
	for _, img := range imgArr.Items {
		if img.Mime == "image/jpeg" {
			imgBuffer, err := downloadImg(img.Link, "image/jpeg")
			if err == nil {
				err = processImg(img, imgBuffer, gameName)
				if err != nil {
					return errors.Wrap(err, "Processing IMG")
				}
				return nil
			}
		}
	}
	itemToInsert := blacklist{
		Game: gameName,
	}
	db.insert("iconblacklists", itemToInsert)
	return nil
}

func downloadImg(URL, imgType string) (*bytes.Buffer, error) {
	imgClient := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := imgClient.Get(URL)
	if err != nil {
		return nil, errors.Wrap(err, "Making Request")
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("Status code was not 200")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Reading Body")
	}
	returnBuffer := new(bytes.Buffer)
	returnBuffer.Write(body)
	isValidImgString := http.DetectContentType(returnBuffer.Bytes())
	isValidImg, _ := regexp.MatchString(imgType, isValidImgString)
	if isValidImg != true {
		return nil, errors.New("Not Valid Img")
	}
	return returnBuffer, nil
}

func getImagesFromGoogle(query string) (googleJSON, error) {
	query = query + " icon"
	imgClient := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := imgClient.Get("https://www.googleapis.com/customsearch/v1?key=" + apiKey + "&cx=" + cseID + "&q=" + url.QueryEscape(query) + "&imgType=photo&searchType=image&fields=items(link,mime)")
	if err != nil {
		return googleJSON{}, errors.Wrap(err, "Making Request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return googleJSON{}, errors.New("Resp Code Not 200")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return googleJSON{}, errors.Wrap(err, "Reading Body")
	}
	var parsedJSON googleJSON
	err = json.Unmarshal(body, &parsedJSON)
	if err != nil {
		return googleJSON{}, errors.Wrap(err, "Parsing JSON")
	}
	return parsedJSON, nil
}
