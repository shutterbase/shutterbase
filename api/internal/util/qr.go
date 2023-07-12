package util

import (
	"bytes"
	"image"
	"image/png"
	"io/ioutil"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

func GetQrCodeString(data []byte) (string, error) {
	reader := qrcode.NewQRCodeReader()

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", err
	}

	result, err := reader.Decode(bitmap, nil)
	if err != nil {
		return "", err
	}

	return result.GetText(), nil
}

/* func GetQrCodeString(data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Error().Err(err).Msg("Error decoding image")
		return "", err
	}
	qrCodes, err := goqr.Recognize(img)
	if err != nil {
		log.Error().Err(err).Msg("Error recognizing qr code")
		return "", err
	}

	if len(qrCodes) == 0 {
		return "", fmt.Errorf("no qr codes found")
	}

	if len(qrCodes) > 1 {
		return "", fmt.Errorf("more than one qr code found")
	}

	result := qrCodes[0]

	return string(result.Payload), nil
} */

func GenerateQrCode(data string) ([]byte, error) {
	writer := qrcode.NewQRCodeWriter()

	img, err := writer.Encode(data, gozxing.BarcodeFormat_QR_CODE, 512, 512, nil)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = png.Encode(buf, img)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(buf)
	if err != nil {
		return nil, err
	}

	return b, nil
}
