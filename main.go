package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
)

const (
	threshold       = 127
	oneSixteenth    = 1 / 16
	threeSixteenths = 3 / 16
	fiveSixteenths  = 5 / 16
	sevenSixteenths = 7 / 16
)

type config struct {
	inFile  string
	outFile string
	invert  bool
}

func init() {
	flag.Usage = func() {
		output := flag.CommandLine.Output()

		fmt.Fprintf(output, "Usage: %s [<flag>...] <image>\n", filepath.Base(os.Args[0]))
		fmt.Fprintln(output, "Image: image file to dither (GIF, JPG, or PNG)")
		fmt.Fprintln(output, "Flags:")

		flag.PrintDefaults()
	}
}

func main() {
	cfg, err := buildConfigFromArgs()

	if err != nil {
		fmt.Fprintln(flag.CommandLine.Output(), err)
		flag.Usage()
		os.Exit(1)
	}

	inImg, inFormat, err := loadImage(cfg.inFile)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	grayImg := imageToGrayscale(inImg)
	errorDiffusionDither(grayImg, cfg.invert)

	if err := saveImage(grayImg, cfg.outFile, inFormat); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildConfigFromArgs() (*config, error) {
	var outFile string
	var invert bool

	flag.StringVar(&outFile, "o", "out", "output image file")
	flag.BoolVar(&invert, "i", false, "invert output image")
	flag.Parse()

	if flag.NArg() == 0 {
		return nil, errors.New("no input image specified")
	}

	inFile := flag.Arg(0)

	if len(filepath.Ext(outFile)) == 0 {
		outFile += filepath.Ext(inFile)
	}

	return &config{
		inFile,
		outFile,
		invert,
	}, nil
}

func loadImage(file string) (image.Image, string, error) {
	reader, err := os.Open(file)

	if err != nil {
		return nil, "", err
	}

	defer reader.Close()

	return image.Decode(reader)
}

func saveImage(img image.Image, file, format string) error {
	writer, err := os.Create(file)

	if err != nil {
		return err
	}

	defer writer.Close()

	switch format {
	case "gif":
		return gif.Encode(writer, img, &gif.Options{NumColors: 256})

	case "jpeg":
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: 100})

	case "png":
		return png.Encode(writer, img)

	default:
		return fmt.Errorf("unknown image format » %s", format)
	}
}

func imageToGrayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	result := image.NewGray(bounds)

	for x := 0; x < bounds.Max.X; x++ {
		for y := 0; y < bounds.Max.Y; y++ {
			result.Set(x, y, result.ColorModel().Convert(img.At(x, y)))
		}
	}

	return result
}

func errorDiffusionDither(img *image.Gray, invert bool) {
	width := img.Bounds().Max.X
	height := img.Bounds().Max.Y
	xLim := width - 1
	yLim := height - 1

	var low uint8 = 0
	var high uint8 = math.MaxUint8

	if invert {
		low, high = high, low
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixVal := img.At(x, y).(color.Gray).Y
			var dithVal uint8

			if pixVal > threshold {
				dithVal = high
			} else {
				dithVal = low
			}

			img.SetGray(x, y, color.Gray{Y: dithVal})

			dithErr := float64(pixVal) - float64(dithVal)

			if x < xLim {
				addErrorContrib(img, x+1, y, dithErr, sevenSixteenths)

				if y < yLim {
					addErrorContrib(img, x+1, y+1, dithErr, oneSixteenth)
				}
			}

			if y < yLim {
				addErrorContrib(img, x, y+1, dithErr, fiveSixteenths)

				if x > 0 {
					addErrorContrib(img, x-1, y+1, dithErr, threeSixteenths)
				}
			}
		}
	}
}

func addErrorContrib(img *image.Gray, x, y int, errVal, perc float64) {
	img.SetGray(x, y, color.Gray{
		Y: uint8(int(img.At(x, y).(color.Gray).Y) + int(errVal*perc)),
	})
}
