package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/generaltso/vibrant"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"gopkg.in/gographics/imagick.v3/imagick"
)

type returnedColors struct {
	Main      vibrant.Color
	Secondary vibrant.Color
}

var pixelWidthBetween = [5]float64{0, 305, 205, 155, 125}
var pixelWidthStart = [5]float64{620, 475, 420, 395, 380}

func getColorPallete(img *image.Image) (returnedColors, error) {
	pallete, err := vibrant.NewPaletteFromImage(*img)
	if err != nil {
		errors.Wrap(err, "Generating Color Pallete")
	}
	colors := pallete.ExtractAwesome()

	if colors["Vibrant"] == nil || colors["DarkMuted"] == nil {
		return getMissingColors(colors), nil
	}
	return returnedColors{
		Main:      colors["Vibrant"].Color,
		Secondary: colors["DarkMuted"].Color,
	}, nil
}

func getMissingColors(colors map[string]*vibrant.Swatch) returnedColors {
	switch len(colors) {
	case 0:
		return returnedColors{}
	case 1:
		for _, swatch := range colors {
			return returnedColors{
				Secondary: swatch.Color,
			}
		}
	default:
		var names []string
		for name := range colors {
			names = append(names, name)
		}
		return returnedColors{
			Main:      colors[names[0]].Color,
			Secondary: colors[names[1]].Color,
		}
	}
	return returnedColors{}
}

func addCircleIcon(img *image.Image, base *imagick.MagickWand) error {
	profilePic := imagick.NewMagickWand()
	defer profilePic.Destroy()
	buffer := new(bytes.Buffer)
	err := png.Encode(buffer, *img)
	if err != nil {
		return errors.Wrap(err, "Loading Image into Wand")
	}
	profilePic.ReadImageBlob(buffer.Bytes())
	profilePic.ResizeImage(450, 450, imagick.FILTER_UNDEFINED)
	height := profilePic.GetImageHeight()
	width := profilePic.GetImageWidth()

	circleMask := imagick.NewMagickWand()
	pw := imagick.NewPixelWand()
	circleDraw := imagick.NewDrawingWand()
	defer pw.Destroy()
	defer circleMask.Destroy()
	defer circleMask.Destroy()
	pw.SetColor("black")
	circleMask.NewImage(height, width, pw)

	pw.SetColor("white")
	circleDraw.SetFillColor(pw)
	circleDraw.Circle(float64(height/2), float64(width/2), float64(height/2), 0)
	circleMask.DrawImage(circleDraw)

	circleMask.SetImageMatte(false)
	profilePic.SetImageMatte(false)
	profilePic.CompositeImage(circleMask, imagick.COMPOSITE_OP_COPY_ALPHA, true, 0, 0)

	base.CompositeImage(profilePic, imagick.COMPOSITE_OP_OVER, true, -90, -120)
	return nil
}

func drawCircles(base *imagick.MagickWand, colors returnedColors, maskToUse string) error {
	mask := imagick.NewMagickWand()
	pw := imagick.NewPixelWand()
	defer mask.Destroy()
	defer pw.Destroy()
	mainImg := imagick.NewMagickWand()
	mask.ReadImage(maskToUse)

	pw.SetColor(colors.Secondary.RGBHex())
	mainImg.NewImage(1000, 1000, pw)

	mainImg.SetImageMatte(false)
	mask.SetImageMatte(false)

	mainImg.CompositeImage(mask, imagick.COMPOSITE_OP_COPY_ALPHA, true, 0, 0)
	base.CompositeImage(mainImg, imagick.COMPOSITE_OP_OVER, true, 0, 0)
	return nil
}

func drawText(base *imagick.MagickWand, name, hoursPlayed, gamesPlayed string, colors returnedColors) error {
	textWand := imagick.NewDrawingWand()
	textColor := imagick.NewPixelWand()
	defer textWand.Destroy()
	defer textColor.Destroy()
	textColor.SetColor(colors.Main.RGBHex())
	textWand.SetFont("main.ttf")
	textWand.SetFillColor(textColor)
	textWand.SetFontSize(70)
	textWand.SetGravity(imagick.GRAVITY_CENTER)

	if len(name) >= 16 {
		name = name[:16] + "\n" + name[16:]
	}

	textWand.Annotation(-345, 0, hoursPlayed+"\nHours\nPlayed")
	textWand.Annotation(-345, 310, gamesPlayed+"\nGames\nPlayed")

	textWand.SetFontSize(100)
	textColor.SetColor(colors.Secondary.RGBHex())
	textWand.SetFillColor(textColor)
	textWand.Annotation(170, -300, name+"\nStats")
	base.DrawImage(textWand)
	return nil
}

func drawBotText(base *imagick.MagickWand, name, totalStats, totalGames, totalImgGenerated, totalServers, totalUsers string, colors returnedColors) error {
	textWand := imagick.NewDrawingWand()
	textColor := imagick.NewPixelWand()
	defer textWand.Destroy()
	defer textColor.Destroy()
	textColor.SetColor(colors.Main.RGBHex())
	textWand.SetFont("main.ttf")
	textWand.SetFillColor(textColor)
	textWand.SetGravity(imagick.GRAVITY_CENTER)

	textWand.SetFontSize(70)
	textWand.Annotation(-345, 0, totalStats+"\nTotal\nStats")
	textWand.Annotation(-345, 310, totalGames+"\nTotal\nGames")
	textWand.Annotation(-40, 0, totalUsers+"\nTotal\nUsers")
	textWand.Annotation(-40, 310, totalServers+"\nTotal\nServers")
	textWand.SetFontSize(90)
	textWand.Annotation(290, 150, totalImgGenerated+"\nImages\nGenerated!")

	textWand.SetFontSize(100)
	textColor.SetColor(colors.Secondary.RGBHex())
	textWand.SetFillColor(textColor)
	textWand.Annotation(170, -300, name+"\nStats")

	base.DrawImage(textWand)
	return nil
}

func addGraph(base *imagick.MagickWand, graphType string, profilePic *image.Image, colors returnedColors, userID string) error {
	graphWand := imagick.NewMagickWand()
	iconWand := imagick.NewMagickWand()
	iconDrawingWand := imagick.NewDrawingWand()
	defer graphWand.Destroy()
	defer iconWand.Destroy()
	defer iconDrawingWand.Destroy()

	switch graphType {
	case "bar":
		var userStatResults []stat
		var userStatsForGraph []stat
		var iconForGraph = make(map[string]icon)
		var lengthOfGraph int
		db.findAllSort("gamestats", "-hours", bson.M{"id": userID, "ignore": false}, &userStatResults)

		if len(userStatResults) >= 5 {
			lengthOfGraph = 4
		} else {
			lengthOfGraph = len(userStatResults) - 1
		}
		for i, item := range userStatResults {
			if i == 5 {
				break
			}
			var currentIcon icon
			db.findOne("gameicons", bson.M{"game": item.Game}, &currentIcon)
			iconForGraph[currentIcon.Game] = currentIcon
			userStatsForGraph = append(userStatsForGraph, item)

			iconWand.ReadImage(path.Join(dataDir, currentIcon.Location))
			iconWand.ResizeImage(100, 100, imagick.FILTER_UNDEFINED)
			var whereToDraw float64
			if i == 0 {
				whereToDraw = pixelWidthStart[lengthOfGraph]
			} else {
				whereToDraw = pixelWidthStart[lengthOfGraph] + (pixelWidthBetween[lengthOfGraph] * float64(i))
			}
			iconDrawingWand.Composite(imagick.COMPOSITE_OP_OVER, whereToDraw, 850, 100, 100, iconWand)
		}
		base.DrawImage(iconDrawingWand)

		barChart, err := createBarChart(userStatsForGraph, iconForGraph, profilePic, colors)
		if err != nil {
			return errors.Wrap(err, "Generating Graph")
		}
		graphWand.ReadImageBlob(barChart.Bytes())
		base.CompositeImage(graphWand, imagick.COMPOSITE_OP_OVER, true, 350, 350)
		break
	case "pie":
		var statsToSend []stat
		var iconsToSend = make(map[string]icon)
		db.findAllSort("gamestats", "-hours", bson.M{"id": userID, "ignore": false}, &statsToSend)

		for _, item := range statsToSend {
			var currentIcon icon
			db.findOne("gameicons", bson.M{"game": item.Game}, &currentIcon)
			iconsToSend[currentIcon.Game] = currentIcon
		}

		pieChart, err := createPieChart(statsToSend, iconsToSend)
		if err != nil {
			return errors.Wrap(err, "Generating Pie Chart")
		}
		graphWand.ReadImageBlob(pieChart.Bytes())
		base.CompositeImage(graphWand, imagick.COMPOSITE_OP_OVER, true, 350, 350)
		break
	}
	return nil
}

func createImage(img *image.Image, hoursPlayed, gamesPlayed, name string, graphType string, userID string) (*bytes.Reader, error) {
	var err error
	tempFileDir := os.Getenv("TMPDIR")
	if tempFileDir == "" {
		tempFileDir = "/tmp"
	}
	imagick.Initialize()
	defer imagick.Terminate()
	mainImg := imagick.NewMagickWand()
	defer mainImg.Destroy()
	bgColor := imagick.NewPixelWand()
	defer bgColor.Destroy()

	colors, _ := getColorPallete(img)
	bgColor.SetColor(colors.Main.RGBHex())
	mainImg.NewImage(1000, 1000, bgColor)

	err = drawCircles(mainImg, colors, "normalMask.png")
	err = addCircleIcon(img, mainImg)
	err = drawText(mainImg, name, hoursPlayed, gamesPlayed, colors)
	err = addGraph(mainImg, graphType, img, colors, userID)

	mainImg.SetImageFormat("PNG")
	blobReader := bytes.NewReader(mainImg.GetImageBlob())
	return blobReader, err
}

func createBotImage(profilePic *image.Image, name, totalStats, totalGames, totalImgGenerated, totalServers, totalUsers string) error {
	var err error
	imagick.Initialize()
	defer imagick.Terminate()
	mainImg := imagick.NewMagickWand()
	defer mainImg.Destroy()
	bgColor := imagick.NewPixelWand()
	defer bgColor.Destroy()

	colors, _ := getColorPallete(profilePic)
	bgColor.SetColor(colors.Main.RGBHex())
	mainImg.NewImage(1000, 1000, bgColor)

	err = drawCircles(mainImg, colors, "botMask.png")
	err = addCircleIcon(profilePic, mainImg)
	err = drawBotText(mainImg, name, totalStats, totalGames, totalImgGenerated, totalServers, totalUsers, colors)
	return err
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
