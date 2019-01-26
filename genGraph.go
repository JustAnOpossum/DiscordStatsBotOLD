package main

import (
	"bytes"

	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	chart "github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

func createChart(chartType string, userID string) (*bytes.Buffer, error) {
	switch chartType {
	case "bar":
		var sortedStatResults []gameStatsStruct
		var bars []chart.Value
		// var height int
		// var barWidth int
		// var fontColor drawing.Color
		db.findAllSort("gamestats", "-hours", bson.M{"id": userID}, &sortedStatResults)
		for i := 0; i < 5; i++ {
			var icon gameStatsStruct
			db.findOne("gameicons", bson.M{"game": sortedStatResults[i].Game}, &icon)
			valueToAdd := chart.Value{
				Value: sortedStatResults[i].Hours,
				Label: sortedStatResults[i].Game,
				Style: chart.Style{
					FillColor:   drawing.ColorBlue,
					StrokeColor: drawing.ColorBlue,
				},
			}
			bars = append(bars, valueToAdd)
		}
		return createBarChart(bars, 0, 0, drawing.ColorBlue)
	case "pie":
		break
	}
	return nil, nil
}

func createBarChart(bars []chart.Value, height, barWidth int, fontColor drawing.Color) (*bytes.Buffer, error) {
	barChart := chart.BarChart{
		Height:   500,
		Width:    640,
		BarWidth: 100,
		Background: chart.Style{
			FillColor: drawing.Color{R: 1, G: 1, B: 1, A: 0},
		},
		XAxis: chart.Style{
			FontSize:  25,
			FontColor: drawing.Color{R: 58, G: 61, B: 86, A: 255},
			Show:      true,
		},
		YAxis: chart.YAxis{
			Range: &chart.ContinuousRange{
				Min: 0,
				Max: 56,
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
