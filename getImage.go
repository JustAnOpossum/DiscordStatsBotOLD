package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"github.com/teris-io/shortid"
)

type bingAnswer struct {
	Type            string `json:"_type"`
	Instrumentation struct {
		PageLoadPingURL interface{} `json:"pageLoadPingUrl"`
	} `json:"instrumentation"`
	WebSearchURL          string      `json:"webSearchUrl"`
	TotalEstimatedMatches int         `json:"totalEstimatedMatches"`
	Value                 []bingValue `json:"value"`
	QueryExpansions       []struct {
		Text         string      `json:"text"`
		DisplayText  string      `json:"displayText"`
		WebSearchURL string      `json:"webSearchUrl"`
		SearchLink   string      `json:"searchLink"`
		Thumbnail1   interface{} `json:"thumbnail1"`
	} `json:"queryExpansions"`
	NextOffsetAddCount int `json:"nextOffsetAddCount"`
	PivotSuggestions   []struct {
		Pivot       string `json:"pivot"`
		Suggestions []struct {
			Text         string `json:"text"`
			DisplayText  string `json:"displayText"`
			WebSearchURL string `json:"webSearchUrl"`
			SearchLink   string `json:"searchLink"`
			Thumbnail    struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			} `json:"thumbnail"`
		} `json:"suggestions"`
	} `json:"pivotSuggestions"`
	DisplayShoppingSourcesBadges bool        `json:"displayShoppingSourcesBadges"`
	DisplayRecipeSourcesBadges   bool        `json:"displayRecipeSourcesBadges"`
	SimilarTerms                 interface{} `json:"similarTerms"`
}

type bingValue struct {
	Name               string      `json:"name"`
	DatePublished      string      `json:"datePublished"`
	HomePageURL        interface{} `json:"homePageUrl"`
	ContentSize        string      `json:"contentSize"`
	HostPageDisplayURL string      `json:"hostPageDisplayUrl"`
	Width              int         `json:"width"`
	Height             int         `json:"height"`
	Thumbnail          struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"thumbnail"`
	ImageInsightsToken     string      `json:"imageInsightsToken"`
	InsightsSourcesSummary interface{} `json:"insightsSourcesSummary"`
	ImageID                string      `json:"imageId"`
	AccentColor            string      `json:"accentColor"`
	WebSearchURL           string      `json:"webSearchUrl"`
	ThumbnailURL           string      `json:"thumbnailUrl"`
	EncodingFormat         string      `json:"encodingFormat"`
	ContentURL             string      `json:"contentUrl"`
}

var apiKey = os.Getenv("CSETOKEN")
var cseID = os.Getenv("CSEID")
var bingToken = os.Getenv("BINGTOKEN")

const endpoint = "https://api.cognitive.microsoft.com/bing/v7.0/images/search"

func processImg(mimeType string, imgBuffer *bytes.Buffer, gameName string) error {
	fmt.Fprintln(out, "Got good image")
	shortID, _ := shortid.Generate()
	var ext string
	if mimeType == "jpeg" {
		ext = ".jpg"
	}
	if mimeType == "png" {
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
	R, G, B := imgColors.Main.RGB()
	itemToInsert := icon{
		Game:     gameName,
		Location: "Images/Game/" + fileName,
		R:        R,
		G:        G,
		B:        B,
	}
	db.insert("gameicons", itemToInsert)
	fmt.Fprintln(out, "Inserted IMG")
	return nil
}

func getGameImg(gameName string) error {
	imgArr, err := getImagesFromBing(gameName)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Error Getting Google Images"))
	}
	for _, img := range imgArr.Value {
		if img.EncodingFormat == "png" {
			imgBuffer, err := downloadImg(img.ContentURL, "image/png")
			if err == nil {
				err = processImg(img.EncodingFormat, imgBuffer, gameName)
				if err != nil {
					return errors.Wrap(err, "Processing IMG")
				}
				return nil
			}
		}
	}
	for _, img := range imgArr.Value {
		if img.EncodingFormat == "jpeg" {
			imgBuffer, err := downloadImg(img.ContentURL, "image/jpeg")
			if err == nil {
				err = processImg(img.EncodingFormat, imgBuffer, gameName)
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
	return errors.New("Added Icon To Blacklist")
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

func getImagesFromBing(query string) (bingAnswer, error) {
	query = query + " icon"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return bingAnswer{}, errors.Wrap(err, "Creating Bing Request")
	}

	param := req.URL.Query()
	param.Add("q", query)
	req.URL.RawQuery = param.Encode()
	req.Header.Add("Ocp-Apim-Subscription-Key", bingToken)

	client := new(http.Client)
	client.Timeout = time.Second * 10

	resp, err := client.Do(req)
	if err != nil {
		return bingAnswer{}, errors.Wrap(err, "Making Bing Request")
	}
	if resp.StatusCode != 200 {
		return bingAnswer{}, errors.Wrap(err, "Bing Resp Not 200")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return bingAnswer{}, errors.Wrap(err, "Parsing Bing Body")
	}

	var returnAns bingAnswer
	err = json.Unmarshal(body, &returnAns)
	if err != nil {
		return bingAnswer{}, errors.Wrap(err, "Parsing Bing JSON")
	}

	return returnAns, nil
}

func getTop5Img(userID string) {
	var results []stat
	howManyToLoop := 5

	db.findAllSort("gamestats", "-hours", bson.M{"id": userID}, &results)

	for i := range results {
		if i == howManyToLoop {
			break
		}
		if db.itemExists("gameicons", bson.M{"game": results[i].Game}) == true {
			continue
		}
		err := getGameImg(results[i].Game)
		if err != nil {
			db.removeAll("gamestats", bson.M{"game": results[i].Game})
			howManyToLoop++
		}
	}
}
