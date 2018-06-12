// Repourposed code from CCDC evidence collection tool to post exploitation intel gathering tool

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/disintegration/gift"
	"github.com/kbinani/screenshot"
	"github.com/pwaller/go-hexcolor"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

var (
	outDir = flag.String("outDir", "", "the output dir to write things to, defaults to the current working dir")
	count  = flag.Int("count", 1, "number of screenshots to take")
	delay  = flag.Duration("delay", 0, "add a delay to each scan")
	//keylog = flag.Bool("keylog", false, "Start a keylogger to record keystrokes")
	wg sync.WaitGroup
)

func main() {
	flag.Parse()
	wg.Add(1)
	os.Chdir(*outDir)
	go screenCapper()
	wg.Wait()
}

func screenCapper() {
	defer wg.Done()
	myIP := getLocalIP()
	for z := 0; z < *count; z++ {
		takeScreenCap(myIP)
		time.Sleep(time.Duration(*delay))
	}
}

func getLocalIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

func takeScreenCap(myIP net.IP) {
	n := screenshot.NumActiveDisplays()
	// Takes a screenshot for each display per call
	for i := 0; i < n; i++ {
		// Take screenshot
		bounds := screenshot.GetDisplayBounds(i)
		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			panic(err)
		}
		// Get watermark data
		rawTime := time.Now()
		curTime := rawTime.Unix()
		timeText := rawTime.Local()
		// Create watermark
		watermarkText := fmt.Sprintf("%s - %s", timeText, myIP.To4().String())
		rawFilepath := fmt.Sprintf("%d_%d_%dx%d.png", curTime, i, bounds.Dx(), bounds.Dy())
		format := "png"
		watermark := createWatermark(watermarkText, 2.0, parseColor("#FF0000FF"))
		sourceBounds := img.Bounds()
		watermarkBounds := watermark.Bounds()
		markedImage := image.NewRGBA(sourceBounds)
		draw.Draw(markedImage, sourceBounds, img, image.ZP, draw.Src)
		var offset image.Point
		for offset.X = watermarkBounds.Max.X / -2; offset.X < sourceBounds.Max.X; offset.X += watermarkBounds.Max.X {
			for offset.Y = watermarkBounds.Max.Y / -2; offset.Y < sourceBounds.Max.Y; offset.Y += watermarkBounds.Max.Y {
				draw.Draw(markedImage, watermarkBounds.Add(offset), watermark, image.ZP, draw.Over)
			}
		}
		fullFilePath := fmt.Sprintf("%s", rawFilepath)
		file, _ := os.Create(fullFilePath)
		defer file.Close()
		switch format {
		case "png":
			err = png.Encode(file, markedImage)
		case "gif":
			err = gif.Encode(file, markedImage, &gif.Options{NumColors: 265})
		case "jpeg":
			err = jpeg.Encode(file, markedImage, &jpeg.Options{Quality: jpeg.DefaultQuality})
		default:
			//log.Fatalf("unknown format %s", format)
		}
		if err != nil {
			//log.Fatalf("unable to encode image: %s", err)
		}
		//fmt.Printf("[*] Screenshot Written: %s\n", rawFilepath)
	}

}

func parseColor(str string) color.Color {
	r, g, b, a := hexcolor.HexToRGBA(hexcolor.Hex(str))
	return color.RGBA{
		A: a,
		R: r,
		G: g,
		B: b,
	}
}

func createWatermark(text string, scale float64, textColor color.Color) image.Image {
	var padding float64 = 2
	w := 8 * (float64(len(text)) + (padding * 2))
	h := 16 * padding
	img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	point := fixed.Point26_6{fixed.Int26_6(64 * padding), fixed.Int26_6(h * 64)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: inconsolata.Regular8x16,
		Dot:  point,
	}
	d.DrawString(text)
	bounds := img.Bounds()
	scaled := image.NewRGBA(image.Rect(0, 0, int(float64(bounds.Max.X)*scale), int(float64(bounds.Max.Y)*scale)))
	draw.BiLinear.Scale(scaled, scaled.Bounds(), img, bounds, draw.Src, nil)
	g := gift.New(
		gift.Rotate(45, color.Transparent, gift.CubicInterpolation),
	)
	rot := image.NewNRGBA(g.Bounds(scaled.Bounds()))
	g.Draw(rot, scaled)
	return rot
}
