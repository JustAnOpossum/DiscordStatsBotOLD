package main

import "gopkg.in/gographics/imagick.v3/imagick"

func main() {
	imagick.Initialize()
	defer imagick.Terminate()

	circleDW := imagick.NewDrawingWand()
	whiteBGPW := imagick.NewPixelWand()
	circleMW := imagick.NewMagickWand()
	whiteBGPW.SetColor("white")
	circleMW.NewImage(1000, 1000, whiteBGPW)

	circleDW.Circle(500, 500, 500, 0)
	circleMW.DrawImage(circleDW)

	profileImgDW := imagick.NewMagickWand()
	//addCircleDW := imagick.NewDrawingWand()
	profileImgDW.ReadImage("test2.png")
	profileImgDW.CompositeImage(circleMW, imagick.COMPOSITE_OP_COPY_ALPHA, false, 0, 0)
	profileImgDW.WriteImage("test.png")
}
