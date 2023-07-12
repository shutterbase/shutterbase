package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"os"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

func main() {
	// open and decode image file
	file, _ := os.Open("time-code-cropped.JPG")
	img, _, _ := image.Decode(file)

	// prepare BinaryBitmap
	bmp, _ := gozxing.NewBinaryBitmapFromImage(img)

  // write image to file
  

	// decode image
	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(result)
}
