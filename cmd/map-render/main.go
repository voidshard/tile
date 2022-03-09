package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/voidshard/tile"

	"github.com/alecthomas/kong"
)

const desc = `Generates a tmx map from an 'infinite map' database file.`

var cli struct {
	// where to find input database file
	Input  string `short:"i" help:"input inifinite map database file (required)"`
	Output string `short:"o" help:"where to write output .tmx map. Defaults to input + coords + .tmx. Overwrites output file if it exists."`

	// how wide/high each tile image should be in pixels
	TileWidth  uint `default:"32" help:"width of each tile in px"`
	TileHeight uint `default:"32" help:"height of each tile in px"`

	X0 int `default:"0" help:"x coord of map, top left corner"`
	Y0 int `default:"0" help:"y coord of map, top left corner"`
	X1 int `default:"0" help:"x coord of map, bottom right corner"`
	Y1 int `default:"0" help:"y coord of map, bottom right corner"`

	// set properties on map
	Props map[string]string `short:"p" help:"set props on resulting map"`
}

func main() {
	kong.Parse(&cli, kong.Name("map-render"), kong.Description(desc))

	if cli.Output == "" {
		cli.Output = fmt.Sprintf("%s_%d.%d_%d.%d.tmx", cli.Input, cli.X0, cli.Y0, cli.X1, cli.Y1)
	}

	if !fileExists(cli.Input) {
		panic(fmt.Sprintf("input file not found: %s", cli.Input))
	}

	inf, err := tile.OpenInfiniteMap(cli.Input)
	if err != nil {
		panic(err)
	}

	m, err := inf.Map(cli.TileWidth, cli.TileHeight, cli.X0, cli.Y0, cli.X1, cli.Y1)
	if err != nil {
		panic(err)
	}

	props := parseProps()
	m.SetMapProperties(props)

	err = m.WriteFile(cli.Output)
	if err != nil {
		panic(err)
	}

	fmt.Println("wrote %s", cli.Output)
}

// fileExists checks if file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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
