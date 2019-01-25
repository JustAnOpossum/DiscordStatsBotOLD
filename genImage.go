package main

import (
	"image"

	"github.com/generaltso/vibrant"
	"github.com/pkg/errors"
	"gopkg.in/gographics/imagick.v3/imagick"
)

type returnedColors struct {
	Main      vibrant.Color
	Secondary vibrant.Color
}

func getColorPallete(img image.Image) (returnedColors, error) {
	pallete, err := vibrant.NewPaletteFromImage(img)
	if err != nil {
		errors.Wrap(err, "Generating Color Pallete")
	}
	colors := pallete.ExtractAwesome()

	if colors["Vibrant"] == nil || colors["Muted"] == nil {
		return getMissingColors(colors), nil
	}
	return returnedColors{
		Main:      colors["Vibrant"].Color,
		Secondary: colors["Muted"].Color,
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
	mask1 := imagick.NewMagickWand()
	defer mask1.Destroy()
	mask2 := imagick.NewMagickWand()
	defer mask2.Destroy()
	profilePic := imagick.NewMagickWand()
	defer profilePic.Destroy()
	circleProfilePic := imagick.NewMagickWand()
	defer circleProfilePic.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()
	drawProfilePic := imagick.NewDrawingWand()
	defer drawProfilePic.Destroy()
	profilePicDW := imagick.NewDrawingWand()
	defer profilePicDW.Destroy()
	err := mask1.ReadImage("secondMask.png")
	if err != nil {
		return errors.Wrap(err, "Reading Mask 2")
	}
	err = mask2.ReadImage("mainMask.png")
	if err != nil {
		return errors.Wrap(err, "Reading Mask 1")
	}
	profilePic.ReadImage("test2.jpg")
	circleProfilePic.NewImage(1000, 1000, pw)

	mask1.SetImageMatte(false)
	mask2.SetImageMatte(false)
	base.SetImageMatte(false)

	base.CompositeImage(mask1, imagick.COMPOSITE_OP_COPY_ALPHA, true, 0, 0)

	drawProfilePic.Composite(imagick.COMPOSITE_OP_UNDEFINED, -40, -40, 400, 400, profilePic)
	circleProfilePic.DrawImage(drawProfilePic)
	circleProfilePic.CompositeImage(mask2, imagick.COMPOSITE_OP_COPY_ALPHA, true, 0, 0)
	profilePicDW.Composite(imagick.COMPOSITE_OP_UNDEFINED, 0, 0, 0, 0, circleProfilePic)

	base.DrawImage(profilePicDW)
	return nil
}

func drawCircles(base *imagick.MagickWand, colors returnedColors) {
	mask := imagick.NewMagickWand()
	defer mask.Destroy()
	pw := imagick.NewPixelWand()
	defer pw.Destroy()
	mainImg := imagick.NewMagickWand()

	pw.SetColor(colors.Secondary.RGBHex())
	mainImg.NewImage(1000, 1000, pw)

	mainImg.SetImageMatte(false)
	mask.SetImageMatte(false)

	mainImg.CompositeImage(mask)
}

func createImage(img image.Image, userID string) error {
	imagick.Initialize()
	defer imagick.Terminate()
	mainImg := imagick.NewMagickWand()
	defer mainImg.Destroy()
	bgColor := imagick.NewPixelWand()
	defer bgColor.Destroy()

	colors, _ := getColorPallete(img)
	bgColor.SetColor(colors.Main.RGBHex())
	mainImg.NewImage(1000, 1000, bgColor)

	drawCircles(mainImg, colors)
	addCircleIcon(nil, mainImg)
	mainImg.WriteImage("test.png")
	return nil
}
