package main

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/nfnt/resize"

	"github.com/voidshard/tile"
)

const desc = `Generates 'tob' (tile-object) files from larger images, including their required tiles.

A 'tob' is essentially a minimal .tmx (doc.mapeditor.org/en/stable/) XML file that lays out how a set of images
('tiles') can be placed together to form an object. That is, for objects made out of multiple tiles the .tmx 
file outlines how tiles are placed in relation to each other (including z-layers & metadata) in order to be 
rendered nicely on to larger objects (or maps).

Each tob includes it's own tileset (images it needs) and tile layers named after the z level the tiles 
should be placed on (a uint). We can thus merge tob(s) into larger maps by merging their tilesets & tile layers
to produced final composite .tmx map files.`

var cli struct {
	// input image to cut this object out from
	Input string `short:"i" help:"input image"`

	// name of output images and tmx tob
	Name string `short:"n" default:"out" help:"output name"`

	// tell us it's ok to overwrite existing stuff (default: no)
	Overwrite bool `help:"overwrite existing file(s) if found"`

	// how wide/high each pixel should be in pixels
	TileWidth  int `default:"32" help:"width of each tile in px"`
	TileHeight int `default:"32" help:"height of each tile in px"`

	// the lowest z level (z-level of the bottom most tiles)
	ZBottom int `short:"b" default:"0" help:"z level of lowest tile"`

	// how the tiles should be spread over z levels in tiles from the bottom up
	ZLayers []int `short:"z" help:"break image into multiple z-layers by y-coords (from bottom)"`

	// invert z-layers to we consider higher layers at the bottom of the obj
	Invert bool `help:"invert z so higher layers are at the bottom (essentially inverts ZLayers to be from the top)"`

	// don't write anything
	DryRun bool `help:"print out what you're planning"`

	// where the desired object lives (rectangle x0,y0 x1,y1 top-left -> bottom-right)
	X0 int    `arg default:"0" help:"where to start getting tiles from (x0)"`
	Y0 int    `arg default:"0" help:"where to start getting tiles from (y0)"`
	X1 string `arg default:"1t" help:"where to stop getting tiles from (x1). Either an absolute value (pixels) or a 't' value (offset in tiles), defaults to 1 tile width (1t)"`
	Y1 string `arg default:"1t" help:"where to stop getting tiles from (y1). Either an absolute value (pixels) or a 't' value (offset in tiles), defaults to 1 tile height (1t)"`

	// set properties on all tiles
	Props map[string]string `short:"p" help:"set props on resulting tob"`

	// multiply z levels by this to get a final z level. We do this to leave space between levels
	// to add features / whatever later by hand if needed.
	// (That is, since we have unused layer numbers between where we draw things we can add tiles
	// between existing layers later on without altering things)
	Mult int `help:"gap between z levels (leave space for future object layers)" default:"10"`

	ImageOnly bool `help:"only cut out image(s) (not .tmx file needed)"`
}

func decode(in io.Reader) (image.Image, error) {
	decoders := []func(io.Reader) (image.Image, error){
		png.Decode,
		gif.Decode,
		jpeg.Decode,
	}

	var lastErr error
	for _, decoder := range decoders {
		im, err := decoder(in)
		if err == nil {
			return im, nil
		}
		lastErr = err
	}

	return nil, lastErr
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

// fileExists checks if file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// sizeToTiles forces input image to be of a width, height of some multiple(s)
// of given input tx,ty (tile x,y size in pixels).
// We default to 1 tile high/wide. Image will be resized to the nearest full
// tile (resized either up or down)
func sizeToTiles(in image.Image, tx, ty int) image.Image {
	width := in.Bounds().Max.X - in.Bounds().Min.X
	height := in.Bounds().Max.Y - in.Bounds().Min.Y

	fitx := width / tx
	fity := height / ty

	// if we're more than half a tile short, make the image bigger
	// to fit, otherwise we'll resize downwards, shrinking the image
	if width%tx > tx/2 {
		fitx++
	}
	if height%ty > ty/2 {
		fity++
	}

	if fitx < 1 {
		fitx = 1
	}
	if fity < 1 {
		fity = 1
	}

	return resize.Resize(
		uint(fitx*tx),
		uint(fity*ty),
		in,
		resize.Lanczos3,
	)
}

// cutOut the rectangle marked by `r` from the given image
func cutOut(in image.Image, r image.Rectangle) image.Image {
	out := image.NewRGBA(image.Rect(0, 0, r.Max.X-r.Min.X, r.Max.Y-r.Min.Y))

	for dx := r.Min.X; dx < r.Max.X; dx++ {
		for dy := r.Min.Y; dy < r.Max.Y; dy++ {
			c := in.At(dx, dy)
			out.Set(dx-r.Min.X, dy-r.Min.Y, c)
		}
	}

	return out
}

// parseProps reads given cli -P --props into a final *Properties
func parseProps() *tile.Properties {
	p := tile.NewProperties()

	for k, v := range cli.Props {
		if v == "true" {
			p.SetBool(k, true)
			continue
		} else if v == "false" {
			p.SetBool(k, false)
			continue
		}

		i, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			p.SetInt(k, int(i))
		} else {
			p.SetString(k, v)
		}
	}

	return p
}

// parseOffset handles reading
// "<someint>t" as "<x/y> + offset in tiles"
// or an absolute value
func parseOffset(tilesize, start int, offset string) int {
	intiles := strings.HasSuffix(offset, "t")

	num, err := strconv.ParseInt(strings.ReplaceAll(offset, "t", ""), 10, 64)
	if err != nil {
		panic(err)
	}

	if intiles {
		return start + (tilesize * int(num))
	}
	return int(num)
}

func main() {
	kong.Parse(
		&cli,
		kong.Name("tob"),
		kong.Description(desc),
	)

	imgdata, err := ioutil.ReadFile(cli.Input)
	if err != nil {
		panic(err)
	}

	in, err := decode(bytes.NewBuffer(imgdata))
	if err != nil {
		panic(err)
	}

	X1 := parseOffset(cli.TileWidth, cli.X0, cli.X1)
	Y1 := parseOffset(cli.TileHeight, cli.Y0, cli.Y1)

	in = cutOut(in, image.Rect(cli.X0, cli.Y0, X1, Y1))

	// resize to fit our desired tile width/height (to closest multiple)
	in = sizeToTiles(in, cli.TileWidth, cli.TileHeight)

	// figure out how many tiles we've got
	width := (in.Bounds().Max.X - in.Bounds().Min.X) / cli.TileWidth
	height := (in.Bounds().Max.Y - in.Bounds().Min.Y) / cli.TileHeight

	props := parseProps()

	fmt.Printf("read (%d,%d)->(%d,%d) from %s ", cli.X0, cli.Y0, X1, Y1, cli.Input)
	fmt.Printf("resize to %dx%d (tiles), making %d new tiles.\n", width, height, width*height)
	fmt.Printf("tile properties: %v\n", props)

	if cli.DryRun {
		fmt.Printf("dry-run detected: doing nothing")
		return
	}

	m := tile.New(&tile.Config{
		MapWidth:   uint(width),
		MapHeight:  uint(height),
		TileWidth:  uint(cli.TileWidth),
		TileHeight: uint(cli.TileHeight),
	})

	numtiles := 0
	for y := 0; y < height; y++ { // for each tile row
		z := 0
		for _, i := range cli.ZLayers {
			if cli.Invert {
				// tiles on the lower rows are considered higher
				if i > y {
					break
				}
			} else {
				// tiles on the lower rows are considered lower
				if i > (height - 1 - y) {
					break
				}
			}
			z += 1
		}
		z *= cli.Mult
		z += cli.ZBottom

		for x := 0; x < width; x++ { // for each tile column
			t := image.NewRGBA(image.Rect(0, 0, cli.TileWidth, cli.TileHeight))
			for ty := 0; ty < cli.TileHeight; ty++ {
				for tx := 0; tx < cli.TileWidth; tx++ {
					c := in.At(tx+x*cli.TileWidth, ty+y*cli.TileHeight)
					t.Set(tx, ty, c)
				}
			}

			// decide image name
			fname := fmt.Sprintf("%s.%d.%d.%d.png", cli.Name, x, y, z)

			// save image
			if fileExists(fname) && !cli.Overwrite {
				fmt.Println("skipping", fname, "exists")
			} else {
				err = savePng(fname, t)
				if err != nil {
					panic(err)
				}
			}

			// set map src & properties
			m.Set(x, y, z, fname)
			m.SetProperties(fname, props)
			numtiles++
		}
	}
	if cli.ImageOnly {
		fmt.Printf("skipping %s.tmx --image-only supplied\n", cli.Name)
		return
	}

	// finally, output the tmx tob
	if fileExists(fmt.Sprintf("%s.tmx", cli.Name)) && !cli.Overwrite {
		fmt.Printf("skipping %s.tmx exists\n", cli.Name)
		return
	}
	f, err := os.OpenFile(fmt.Sprintf("%s.tmx", cli.Name), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	err = m.Encode(f)
	if err != nil {
		panic(err)
	}

}
