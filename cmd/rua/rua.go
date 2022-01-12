// rua
// thx: https://go.dev/blog/image-draw, https://pkg.go.dev/image
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"strings"

	_ "embed"

	xdraw "golang.org/x/image/draw"
)

var (
	flagHelp    = flag.Bool("h", false, "Shows usage options.")
	flagProfile = flag.String("p", "", "choose profile photo")
	flagFat     = flag.Int("f", 5, "set fat")
	flagSpeed   = flag.Int("s", 8, "set speed")
	flagX       = flag.Int("x", 0, "profile photo point.X")
	flagY       = flag.Int("y", 0, "profile photo point.Y")
)

func banner() {
	t := `
▄▄▄  ▄• ▄▌ ▄▄▄· 
▀▄ █·█▪██▌▐█ ▀█ 
▐▀▀▄ █▌▐█▌▄█▀▀█ 
▐█•█▌▐█▄█▌▐█ ▪▐▌
.▀  ▀ ▀▀▀  ▀  ▀ 
`
	fmt.Println(t)
}

//go:embed template.gif
var templateGIF []byte

func main() {
	banner()
	flag.Parse()
	if *flagHelp || *flagProfile == "" {
		fmt.Printf("Usage: `rua` [options]\n\n")
		flag.PrintDefaults()
		return
	}
	if *flagProfile == "" {
		log.Fatalln("Choose profile photo!")
	}

	if err := ruax(*flagProfile); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("rua star star star!")
}

func ruax(pfn string) error {
	var (
		tmplgif  *gif.GIF
		err      error
		palettes color.Palette
	)
	palettes = color.Palette{color.Transparent, color.White}
	palettes = append(palettes, palette.WebSafe...)

	tmplgif, err = gif.DecodeAll(bytes.NewReader(templateGIF[:]))
	if err != nil {
		return fmt.Errorf("decode 'template.gif' error: %s", err)
	}
	config, err := gif.DecodeConfig(bytes.NewReader(templateGIF[:]))
	if err != nil {
		return fmt.Errorf("decode 'template.gif' config error: %s", err)
	}

	rect := image.Rect(0, 0, config.Width, config.Height)

	profilef, err := os.Open(pfn)
	if err != nil {
		return err
	}
	defer profilef.Close()
	var profileImg image.Image
	if strings.HasSuffix(pfn, ".png") {
		profileImg, err = png.Decode(profilef)
	} else if strings.HasSuffix(pfn, ".jpg") || strings.HasSuffix(pfn, ".jpeg") {
		profileImg, err = jpeg.Decode(profilef)
	} else {
		return errors.New("picture type is not supported")
	}
	if err != nil {
		return err
	}

	x := rect.Max.X - 20 + *flagFat
	y := rect.Max.X * profileImg.Bounds().Dy() / profileImg.Bounds().Dx()
	headRect := image.Rect(0, 0, x, y)

	point := image.Point{headRect.Min.X + *flagX, headRect.Min.Y - 15 + *flagY}

	newimgs := make([]*image.Paletted, 0)
	j := 0
	for i, srcimg := range tmplgif.Image {
		if i%2 != 0 {
			continue
		}
		// TODO 动态的大小
		j++
		switch j {
		case 1:
		case 2:
			headRect.Min.X -= 5
			headRect.Max.X += 5
			headRect.Min.Y += 5
			headRect.Max.Y -= 5
			point.Y -= 5
		case 3:
			headRect.Min.X -= 5
			headRect.Max.X += 5
			headRect.Min.Y += 5
			headRect.Max.Y -= 5
			point.Y -= 5
		case 4:
			headRect.Min.X += 5
			headRect.Max.X -= 5
			headRect.Min.Y -= 5
			headRect.Max.Y += 5
			point.Y += 5
		}

		head := image.NewRGBA(headRect)
		xdraw.NearestNeighbor.Scale(head, head.Rect, profileImg, profileImg.Bounds(), draw.Over, nil)

		out := image.NewPaletted(rect, palettes)
		draw.Draw(out, out.Bounds(), head, point, draw.Src)
		draw.Draw(out, out.Bounds(), srcimg, image.Point{}, draw.Over)
		newimgs = append(newimgs, out)
	}

	output, err := os.Create(fmt.Sprintf("%s.gif", strings.Split(pfn, ".")[0]))
	if err != nil {
		return err
	}
	defer output.Close()

	delay := make([]int, len(newimgs))
	for i := range delay {
		delay[i] = *flagSpeed
	}
	disposal := make([]byte, len(newimgs))
	for i := range disposal {
		disposal[i] = 2
	}

	newgif := &gif.GIF{
		Image:     newimgs,
		Delay:     delay,
		LoopCount: 0,
		Disposal:  disposal,
	}
	return gif.EncodeAll(output, newgif)
}
