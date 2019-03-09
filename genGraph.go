package main

import (
	"bytes"
	"fmt"
	"image"
	"math"

	"github.com/pkg/errors"
	chart "github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
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
		roundedHours := math.Round(value.Hours)
		valueToAdd := chart.Value{
			Value: value.Hours,
			Label: fmt.Sprintf("%g", roundedHours),
			Style: chart.Style{
				FillColor:   drawing.Color{R: uint8(currentIcon.R), G: uint8(currentIcon.G), B: uint8(currentIcon.B), A: 255},
				StrokeColor: drawing.Color{R: uint8(currentIcon.R), G: uint8(currentIcon.G), B: uint8(currentIcon.B), A: 255},
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

	for _, item := range userStats {
		valueToAdd := chart.Value{
			Label: item.Game,
			Value: item.Hours,
			Style: chart.Style{
				FillColor: drawing.Color{R: uint8(icons[item.Game].R), G: uint8(icons[item.Game].G), B: uint8(icons[item.Game].B), A: 255},
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
