// gif2img
// thx: https://gist.github.com/linw1995/820b05ac3fbab34937635d30a61740fb
package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"os"
	"strings"
)

var (
	flagHelp   = flag.Bool("h", false, "Shows usage options.")
	flagTarget = flag.String("f", "", "choose gif file")
)

func banner() {
	t := `
       _    ___ ______  _             
      (_)  / __|_____ \(_)            
  ____ _ _| |__  ____) )_ ____   ____ 
 / _  | (_   __)/ ____/| |    \ / _  |
( (_| | | | |  | (_____| | | | ( (_| |
 \___ |_| |_|  |_______)_|_|_|_|\___ |
(_____|                        (_____|
`
	fmt.Println(t)
}

func main() {
	banner()
	flag.Parse()
	if *flagHelp || *flagTarget == "" {
		fmt.Printf("Usage: gif2img [options]\n\n")
		flag.PrintDefaults()
		return
	}

	inGif, err := decodeGif(*flagTarget)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := os.Open(*flagTarget)
	if err != nil {
		log.Fatalf("open '%s' error: %s", *flagTarget, err)
	}
	defer f.Close()
	config, err := gif.DecodeConfig(f)
	if err != nil {
		fmt.Printf("decode gif config '%s' error: %s", *flagTarget, err)
		return
	}

	rect := image.Rect(0, 0, config.Width, config.Height)
	if rect.Min == rect.Max {
		var max image.Point
		for _, frame := range inGif.Image {
			maxF := frame.Bounds().Max
			if max.X < maxF.X {
				max.X = maxF.X
			}
			if max.Y < maxF.Y {
				max.Y = maxF.Y
			}
		}
		rect.Max = max
	}

	format := fmt.Sprintf("%s-%%d.png", strings.Split(*flagTarget, ".")[0])
	for i, srcimg := range inGif.Image {
		img := image.NewRGBA(rect)
		subfn := fmt.Sprintf(format, i)
		f, err := os.Create(subfn)
		if err != nil {
			fmt.Printf("create file '%s' error: %s", subfn, err)
			return
		}
		draw.Draw(img, img.Bounds(), srcimg, srcimg.Rect.Min, draw.Src)
		png.Encode(f, img)
		f.Close()

		fmt.Printf("[+] '%s' ok...\n", subfn)
	}
}

func decodeGif(fn string) (*gif.GIF, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("open '%s' error: %s", fn, err)
	}
	defer f.Close()
	Gif, err := gif.DecodeAll(f)
	if err != nil {
		return nil, fmt.Errorf("decode '%s' error: %s", fn, err)
	}
	return Gif, nil
}
