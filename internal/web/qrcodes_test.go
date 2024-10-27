package web

import (
	"bytes"
	"image"
	"testing"
)

func TestServerUrl(t *testing.T) {
	data := map[string]string{
		"localhost:8080": "http://localhost:8080",
		"my.custom.url":  "https://my.custom.url",
	}
	for input, expected := range data {
		output := server(input)
		if output != expected {
			t.Errorf("Unexpected output for %s: %s", input, output)
		}
	}
}

func TestMakeQrCode(t *testing.T) {
	var buf bytes.Buffer

	writeQrCode("servername", "12345", "statement", &buf)

	image, imgType, err := image.Decode(&buf)
	if err != nil {
		t.Errorf("Error decoding image: %v", err)
	}
	if imgType != "png" {
		t.Errorf("Unexpected image type: %s", imgType)
	}
	if image.Bounds().Max.X != 320 {
		t.Errorf("Unexpected image size: %d", image.Bounds().Max.X)
	}
	if image.Bounds().Max.X != image.Bounds().Max.Y {
		t.Error("Image is not square")
	}
}
