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

var barWidthConst = [5]int{500, 250, 188, 141, 113}

func createBarChart(userStats []stat, icons map[string]icon, profilePic *image.Image, profilePicColors returnedColors) (*bytes.Buffer, error) {
	var bars []chart.Value
	var height float64
	var fontColor drawing.Color
	if len(userStats) == 0 {
		return new(bytes.Buffer), nil
	}
	height = userStats[0].Hours
	for _, value := range userStats {
		var currentIcon = icons[value.Game]
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
	R, G, B := profilePicColors.Secondary.RGB()
	fontColor = drawing.Color{R: uint8(R), G: uint8(G), B: uint8(B), A: 255}

	barChart := chart.BarChart{
		Height:   500,
		Width:    640,
		BarWidth: barWidthConst[len(bars)-1],
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

func createPieChart(userStats []stat, icons map[string]icon) (*bytes.Buffer, error) {
	if len(userStats) == 0 {
		return new(bytes.Buffer), nil
	}

	var valuesToAdd []chart.Value
	totalHours := db.countHours(bson.M{"id": userStats[0].ID})

	for _, item := range userStats {
		var label string
		parseHex, err := colors.ParseHEX(icons[item.Game].Color)
		if err != nil {
			return nil, errors.Wrap(err, "Parsing Hex")
		}
		if item.Hours/totalHours > 0.10 {
			label = item.Game
		}
		RGB := parseHex.ToRGB()
		valueToAdd := chart.Value{
			Label: label,
			Value: item.Hours,
			Style: chart.Style{
				FillColor: drawing.Color{R: uint8(RGB.R), G: uint8(RGB.G), B: uint8(RGB.B), A: 255},
				FontSize:  20,
			},
		}
		valuesToAdd = append(valuesToAdd, valueToAdd)
	}

	pie := chart.PieChart{
		Height: 600,
		Width:  600,
		Background: chart.Style{
			FillColor: drawing.Color{R: 1, G: 1, B: 1, A: 0},
		},
		Canvas: chart.Style{
			FillColor: drawing.Color{R: 1, G: 1, B: 1, A: 0},
		},
		Values: valuesToAdd,
	}

	pieChart := new(bytes.Buffer)
	err := pie.Render(chart.PNG, pieChart)
	if err != nil {
		return nil, errors.Wrap(err, "Rendering Pie Chart")
	}
	return pieChart, nil
}
