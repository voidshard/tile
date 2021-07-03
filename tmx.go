/* this file is a simplified set of structs for reading & writing TMX files.

Much of this code was lifted from github.com/bcvery1/tilepix including
the encode / decode functions (all credit to authors).

We only need a small part of the feature set of TMX in order to do what we want
so we only bother to parse / write those things.
*/
package tile

import (
	"encoding/xml"
	"strconv"
	"strings"
)

// Map is a TMX file structure representing the map as a whole.
// We support only a subset of TMX (read: the bits that we actually use).
// - we only care about one tileset (that we add to when we add objects if needed)
// - we use CSV tile data encoding minus compression (we can compress the file in transit & at rest anyways)
// - we stick to the 'orthogonal' orientation
type Map struct {
	XMLName        xml.Name      `xml:"map"`              // sets top level xml name
	Orientation    string        `xml:"orientation,attr"` // we only support "orthogonal"
	Width          int           `xml:"width,attr"`       // in tiles
	Height         int           `xml:"height,attr"`      // in tiles
	TileWidth      int           `xml:"tilewidth,attr"`   // in pixels
	TileHeight     int           `xml:"tileheight,attr"`  // in pixels
	RootProperties []*Property   `xml:"properties>property"`
	Tilesets       []*Tileset    `xml:"tileset"`
	ImageLayers    []*ImageLayer `xml:"imagelayer"`
	TileLayers     []*TileLayer  `xml:"layer"`
	nextID         uint
}

// newTilelayer creates a new tilelayer with the given name &
// adds it to the map
func (m *Map) newTilelayer(name string) *TileLayer {
	l := &TileLayer{
		Name:       name,
		Width:      m.Width,
		Height:     m.Height,
		Properties: []*Property{},
		Data: Data{
			Encoding:    "csv",
			Compression: "",
			RawData:     []byte{},
		},
		decodedTiles: make([]uint, m.Width*m.Height),
	}
	m.TileLayers = append(m.TileLayers, l)
	return l
}

// newTile registers a new tile by it's image.
// We also
// - add the tile to the last tileset
// - set internal caches for finding the tile
func (m *Map) newTile(source string) *Tile {
	t := &Tile{
		ID:         m.nextID,
		Image:      &Image{Source: source, Width: m.TileWidth, Height: m.TileHeight},
		Properties: []*Property{},
	}
	ts := m.Tilesets[len(m.Tilesets)-1]
	ts.Tiles = append(ts.Tiles, t)
	ts.tileByID[t.ID] = t
	ts.tileBySrc[source] = t
	m.nextID++
	return t
}

// newTileset makes a new tileset starting at `first`
func newTileset(name string, first uint) *Tileset {
	return &Tileset{
		FirstGID:   first,
		Name:       name,
		Properties: []*Property{},
		Tiles:      []*Tile{},
		tileByID:   map[uint]*Tile{},
		tileBySrc:  map[string]*Tile{},
	}
}

// ImageLayer is a TMX file structure which references an image layer, with associated properties.
type ImageLayer struct {
	ID    uint   `xml:"id,attr"`
	Name  string `xml:"name,attr"`
	Image *Image `xml:"image"`
}

// Tileset is a TMX file structure which represents a Tiled Tileset
type Tileset struct {
	FirstGID   uint        `xml:"firstgid,attr"`
	Name       string      `xml:"name,attr"`
	TileWidth  int         `xml:"tilewidth,attr"`
	TileHeight int         `xml:"tileheight,attr"`
	Properties []*Property `xml:"properties>property"`
	Tiles      []*Tile     `xml:"tile"`
	Image      *Image      `xml:"image"`
	tileByID   map[uint]*Tile
	tileBySrc  map[string]*Tile
}

// Property is a TMX file structure which holds a Tiled property.
type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
	Type  string `xml:"type,attr"` // string (default), int, bool + other (we don't use)
}

// Image is an image file in TMX
type Image struct {
	Source string `xml:"source,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
}

// Tile is a TMX tile (from a tileset)
type Tile struct {
	ID         uint        `xml:"id,attr"`
	Image      *Image      `xml:"image"`
	Properties []*Property `xml:"properties>property"`
}

// TileLayer is a TMX file structure which can hold any type of Tiled layer.
type TileLayer struct {
	ID           uint        `xml:"id,attr"`
	Width        int         `xml:"width,attr"`
	Height       int         `xml:"height,attr"`
	Name         string      `xml:"name,attr"`
	Properties   []*Property `xml:"properties>property"` // we support CSV & Base64
	Data         Data        `xml:"data"`
	decodedTiles []uint
}

// Data is a TMX file structure holding data.
type Data struct {
	Encoding    string `xml:"encoding,attr"`
	Compression string `xml:"compression,attr"`
	RawData     []byte `xml:",innerxml"`
}

// encodeCSV turns our list of tile ids back into csv format
func (d *Data) encodeCSV(width, height int, in []uint) ([]byte, error) {
	values := make([]string, height)

	for row := 0; row < height; row++ {
		csvrow := make([]string, width)
		for col := 0; col < width; col++ {
			csvrow[col] = strconv.Itoa(int(in[row*width+col]))
		}
		values[row] = strings.Join(csvrow, ",")
	}

	return []byte("\n" + strings.Join(values, ",\n") + "\n"), nil
}

// decodeCSV reads csv encoded tile data
func (d *Data) decodeCSV() ([]uint, error) {
	cleaner := func(r rune) rune {
		if (r >= '0' && r <= '9') || r == ',' {
			return r
		}
		return -1
	}

	rawDataClean := strings.Map(cleaner, string(d.RawData))

	str := strings.Split(string(rawDataClean), ",")

	gids := make([]uint, len(str))
	for i, s := range str {
		d, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return nil, err
		}
		gids[i] = uint(d)
	}
	return gids, nil
}
