package main

import (
	"bytes"
	"fmt"
	"image"
	"math"

	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	chart "github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	colors "gopkg.in/go-playground/colors.v1"
)

var barWidth = [5]int{500, 250, 188, 141, 113}

func createChart(chartType string, userID string, profilePic *image.Image) (*bytes.Buffer, error) {
	switch chartType {
	case "bar":
		var sortedStatResults []stat
		var bars []chart.Value
		var height float64
		db.findAllSort("gamestats", "-hours", bson.M{"id": userID}, &sortedStatResults)
		if len(sortedStatResults) == 0 {
			return new(bytes.Buffer), nil
		}
		height = sortedStatResults[0].Hours
		for i, value := range sortedStatResults {
			if i == 5 {
				break
			}
			var currentIcon icon
			db.findOne("gameicons", bson.M{"game": value.Game}, &currentIcon)
			hex, err := colors.ParseHEX(currentIcon.Color)
			if err != nil {
				return nil, errors.Wrap(err, "Parsing Hex")
			}
			RGB := hex.ToRGB()
			roundedHours := math.Round(value.Hours)
			valueToAdd := chart.Value{
				Value: value.Hours,
				Label: fmt.Sprintf("%g", roundedHours),
				Style: chart.Style{
					FillColor:   drawing.Color{R: RGB.R, G: RGB.G, B: RGB.B, A: 255},
					StrokeColor: drawing.Color{R: RGB.R, G: RGB.G, B: RGB.B, A: 255},
				},
			}
			bars = append(bars, valueToAdd)
		}
		colors, err := getColorPallete(profilePic)
		if err != nil {
			return nil, errors.Wrap(err, "Getting Profile Pic Colors")
		}
		R, G, B := colors.Secondary.RGB()
		return createBarChart(bars, height, barWidth[len(bars)-1], drawing.Color{R: uint8(R), G: uint8(G), B: uint8(B), A: 255})
	case "pie":
		break
	}
	return nil, errors.New("Something Bad Happened :/")
}

func createBarChart(bars []chart.Value, height float64, barWidth int, fontColor drawing.Color) (*bytes.Buffer, error) {
	barChart := chart.BarChart{
		Height:   500,
		Width:    640,
		BarWidth: barWidth,
		Background: chart.Style{
			FillColor: drawing.Color{R: 1, G: 1, B: 1, A: 0},
		},
		XAxis: chart.Style{
			FontSize:  25,
			FontColor: fontColor,
			Show:      true,
		},
		YAxis: chart.YAxis{
			Range: &chart.ContinuousRange{
				Min: 0,
				Max: height,
			},
		},
		Canvas: chart.Style{
			FillColor: drawing.Color{R: 1, G: 1, B: 1, A: 0},
		},
		Bars: bars,
	}

	var graphImgByteArr []byte
	graphImg := bytes.NewBuffer(graphImgByteArr)
	err := barChart.Render(chart.PNG, graphImg)
	if err != nil {
		return nil, errors.Wrap(err, "Generating Bar Graph")
	}
	return graphImg, nil
}

func createPieChart() {

}
