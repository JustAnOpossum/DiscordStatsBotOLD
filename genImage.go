package main

import (
	"bytes"
	"image"
	"image/png"

	"github.com/generaltso/vibrant"
	"github.com/pkg/errors"
	"gopkg.in/gographics/imagick.v2/imagick"
)

type returnedColors struct {
	Main      vibrant.Color
	Secondary vibrant.Color
}

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
	defer circleMask.Destroy()
	defer pw.Destroy()
	defer circleDraw.Destroy()
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

func drawCircles(base *imagick.MagickWand, colors returnedColors) error {
	mask := imagick.NewMagickWand()
	pw := imagick.NewPixelWand()
	defer mask.Destroy()
	defer pw.Destroy()
	mainImg := imagick.NewMagickWand()
	mask.ReadImage("circleMask.png")

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

func addGraph(base *imagick.MagickWand, graph *bytes.Buffer) error {
	graphWand := imagick.NewMagickWand()
	defer graphWand.Destroy()
	graphWand.ReadImageBlob(graph.Bytes())

	base.CompositeImage(graphWand, imagick.COMPOSITE_OP_OVER, true, 350, 350)
	return nil
}

func createImage(img *image.Image, hoursPlayed, gamesPlayed, name string, graph *bytes.Buffer) error {
	var err error
	imagick.Initialize()
	defer imagick.Terminate()
	mainImg := imagick.NewMagickWand()
	defer mainImg.Destroy()
	bgColor := imagick.NewPixelWand()
	defer bgColor.Destroy()

	colors, _ := getColorPallete(img)
	bgColor.SetColor(colors.Main.RGBHex())
	mainImg.NewImage(1000, 1000, bgColor)

	err = drawCircles(mainImg, colors)
	err = addCircleIcon(img, mainImg)
	err = drawText(mainImg, name, hoursPlayed, gamesPlayed, colors)
	err = addGraph(mainImg, graph)
	mainImg.WriteImage("test.png")
	return err
}
