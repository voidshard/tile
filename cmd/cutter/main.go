package main

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/kong"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
)

const desc = `Removes lines from tiled images via cutting out tiles & re-glueing together.`

var cli struct {
	Input string `short:"i" help:"input image"`

	TileWidth  int `default:"16" help:""`
	TileHeight int `default:"16" help:""`

	LineWidth int `default:"1" help:""`
}

func main() {
	kong.Parse(
		&cli,
		kong.Name("cutter"),
		kong.Description(desc),
	)

	imgdata, err := ioutil.ReadFile(cli.Input)
	if err != nil {
		panic(err)
	}

	in, _, err := image.Decode(bytes.NewBuffer(imgdata))
	if err != nil {
		panic(err)
	}

	bnds := in.Bounds()
	tilesHigh := (bnds.Max.Y - bnds.Min.Y) / (cli.TileHeight + cli.LineWidth)
	tilesWide := (bnds.Max.X - bnds.Min.X) / (cli.TileWidth + cli.LineWidth)

	dst := image.NewRGBA(image.Rect(0, 0, cli.TileWidth*tilesWide, cli.TileHeight*tilesHigh))

	for ty := 0; ty < tilesHigh; ty++ {
		for tx := 0; tx < tilesWide; tx++ {
			drect := image.Rect(tx*cli.TileWidth, ty*cli.TileHeight, (tx+1)*cli.TileWidth, (ty+1)*cli.TileHeight)
			spnt := image.Pt(1+cli.LineWidth+tx*(cli.TileWidth+cli.LineWidth), 2+cli.LineWidth+ty*(cli.TileHeight+cli.LineWidth))
			fmt.Printf("copying %v -> %v\n", spnt, drect)
			draw.Draw(
				dst,
				drect,
				in,
				spnt,
				draw.Src,
			)
		}
	}

	err = savePng(fmt.Sprintf("%s.cut.png", cli.Input), dst)
	if err != nil {
		panic(err)
	}
}

// savePng to disk
func savePng(fpath string, in image.Image) error {
	buff := new(bytes.Buffer)
	err := png.Encode(buff, in)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fpath, buff.Bytes(), 0644)
}
