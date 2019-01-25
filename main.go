package main

import (
	"image"
	_ "image/jpeg"
	"os"
)

func main() {

	file, _ := os.Open("test2.jpg")
	defer file.Close()
	img, _, _ := image.Decode(file)
	createImage(img, "123")

	// imagick.Initialize()
	// defer imagick.Terminate()
	// mw := imagick.NewMagickWand()
	// pw := imagick.NewPixelWand()
	// dw := imagick.NewDrawingWand()
	// pw.SetColor("white")
	// mw.NewImage(1000, 1000, pw)

	// dw.SetFont("test.ttf")
	// dw.Annotation(100, 100, "dasfox stats")
	// mw.DrawImage(dw)
	// mw.WriteImage("test.png")
}
