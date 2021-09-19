/* file adds helper functions to our tmx map wrapper struct.
 */
package tile

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
)

// New returns a new map with defaults set.
func New(cfg *Config) *Map {
	return &Map{
		Orientation:    "orthogonal",
		Width:          int(cfg.MapWidth),
		Height:         int(cfg.MapHeight),
		TileWidth:      int(cfg.TileWidth),
		TileHeight:     int(cfg.TileHeight),
		Tilesets:       []*Tileset{newTileset("default", 1)},
		RootProperties: []*Property{},
		TileLayers:     []*TileLayer{},
		ImageLayers:    []*ImageLayer{},
		nextID:         1,
	}
}

// MapProperties returns properties set on the map itself
func (m *Map) MapProperties() *Properties {
	return newPropertiesFromList(m.RootProperties)
}

// SetMapProperties sets properties on the map
func (m *Map) SetMapProperties(in *Properties) {
	m.RootProperties = in.toList()
}

// Fits returns if copying in the given map to (x,y,zoffset) would
// overwrite an existing tile on any layer in our current map.
func (m *Map) Fits(x, y, zoffset int, o *Map) bool {
	if zoffset < 0 {
		levels := m.ZLevels()
		for i := len(levels) - 1; i >= 0; i-- {
			props := m.At(x, y, i)
			if props != nil {
				zoffset = i
				break
			}
		}
		if zoffset < 0 {
			zoffset = 0
		}
	}

	for _, tl := range o.TileLayers {
		z, err := strconv.ParseInt(tl.Name, 10, 64)
		if err != nil {
			continue
		}

		for index, tid := range tl.decodedTiles {
			if tid == 0 {
				continue // nil tile
			}

			// the reverse of index = y * width + x
			tx := index % o.Width
			ty := index / o.Width

			// check if the object goes off the map
			if tx+x < 0 || tx+x >= m.Width || ty+y < 0 || ty+y >= m.Height {
				return false
			}

			// check if there is a tile there
			t := m.At(tx+x, ty+y, int(z)+zoffset)
			if t != nil {
				return false
			}
		}
	}

	return true
}

// Add the given map `o` starting at the location x,y
// We merge the TileLayers of both maps, but we only consider TileLayers that
// we write ie, those named after their z-layers (0, 1, 2, 3, ...).
// (x,y) is the top left tile, irrespective of z-layer.
func (m *Map) Add(x, y, zoffset int, o *Map) {
	ts := o.Tilesets[0]
	if zoffset < 0 {
		levels := m.ZLevels()
		for i := len(levels) - 1; i >= 0; i-- {
			props := m.At(x, y, i)
			if props != nil {
				zoffset = i
				break
			}
		}
		if zoffset < 0 {
			zoffset = 0
		}
	}

	for _, tl := range o.TileLayers {
		z, err := strconv.ParseInt(tl.Name, 10, 64)
		if err != nil {
			continue
		}

		for index, tid := range tl.decodedTiles {
			if tid == 0 {
				// skip nil tile
				continue
			}
			tile, ok := ts.tileByID[tid]
			if !ok {
				// implies we have a tile with no tileset entry ??
				continue
			}

			// the reverse of index = y * width + x
			tx := index % o.Width
			ty := index / o.Width

			src := tile.Image.Source
			m.Set(tx+x, ty+y, int(z)+zoffset, src)
			m.SetProperties(src, m.Properties(src).Merge(o.Properties(src)))
		}
	}
}

// ZLevels returns all z-level maps (maps named after an int) sorted low -> high.
func (m *Map) ZLevels() []int {
	levels := []int{}
	for _, tl := range m.TileLayers {
		z, err := strconv.ParseInt(tl.Name, 10, 64)
		if err != nil {
			continue
		}

		levels = append(levels, int(z))
	}
	sort.Ints(levels)
	return levels
}

// At returns the properties of the tile at (x, y, z) or nil if not set
// (ie. set to the nil tile).
func (m *Map) At(x, y, z int) *Properties {
	var l *TileLayer
	for _, tl := range m.TileLayers {
		// match z => tilelayer name
		if fmt.Sprintf("%d", z) == tl.Name {
			l = tl
			break
		}
	}
	if l == nil {
		return nil
	}

	index := y*m.Width + x
	if index >= len(l.decodedTiles) || index < 0 {
		return nil
	}

	id := l.decodedTiles[index]
	if id == 0 {
		// the nil tile
		return nil
	}

	for _, ts := range m.Tilesets {
		t, ok := ts.tileByID[id]
		if !ok {
			return nil
		}
		return newPropertiesFromList(t.Properties)
	}

	return nil
}

// Set the tile source for (x,y,z) to some image src.
// If the image doesn't exist in a tileset it is added.
// If "" is passed for source the nil tile is set (ID: 0).
func (m *Map) Set(x, y, z int, source string) error {
	var l *TileLayer
	for _, tl := range m.TileLayers {
		// match z => tilelayer name
		if fmt.Sprintf("%d", z) == tl.Name {
			l = tl
			break
		}
	}

	index := y*m.Width + x
	if l == nil {
		l = m.newTilelayer(fmt.Sprintf("%d", z))
	}

	if len(l.decodedTiles) <= index || index < 0 {
		return fmt.Errorf("index %d is out of bounds for this map", index)
	}

	if source == "" {
		// nil tile
		l.decodedTiles[index] = 0
	}

	var t *Tile
	for _, ts := range m.Tilesets {
		var ok bool
		t, ok = ts.tileBySrc[source]
		if ok {
			break
		}
	}
	if t == nil {
		t = m.newTile(source)
	}
	l.decodedTiles[index] = t.ID
	return nil
}

// SetBackground sets (/creates) an image layer "background" and sets
// it's image to the given src. It's dimensions are also configured
// to cover the map
func (m *Map) SetBackground(src string) {
	var l *ImageLayer
	for _, layer := range m.ImageLayers {
		if layer.Name == "background" {
			l = layer
			break
		}
	}

	if l == nil {
		l = &ImageLayer{
			Name:  "background",
			Image: &Image{},
		}
		m.ImageLayers = append(m.ImageLayers, l)
	}

	l.Image.Source = src
	l.Image.Width = m.TileWidth * m.Width
	l.Image.Height = m.TileHeight * m.Height
}

// Properties returns the properties of the tile indicated by the `source`
// image (or nil).
func (m *Map) Properties(source string) *Properties {
	if source == "" {
		// the nil tile has no properties
		return nil
	}

	var t *Tile
	for _, ts := range m.Tilesets {
		var ok bool
		t, ok = ts.tileBySrc[source]
		if ok {
			break
		}
	}
	if t == nil {
		return nil
	}
	return newPropertiesFromList(t.Properties)
}

// SetProperties sets properties on the tile indicated by the given source
// image.
func (m *Map) SetProperties(source string, in *Properties) {
	if source == "" {
		// cannot set properties on the nil tile
		return
	}

	var t *Tile
	for _, ts := range m.Tilesets {
		var ok bool
		t, ok = ts.tileBySrc[source]
		if ok {
			break
		}
	}
	if t == nil {
		t = m.newTile(source)
	}

	t.Properties = in.toList()
}

// Encode the current map as XML to a io.Writer stream
func (m *Map) Encode(w io.Writer) error {
	for _, ts := range m.Tilesets {
		ts.tileBySrc = map[string]*Tile{}
		ts.tileByID = map[uint]*Tile{}

		for _, t := range ts.Tiles {
			ts.tileBySrc[t.Image.Source] = t
			ts.tileByID[t.ID] = t
		}
	}

	// tiled renders maps in order of ID, low -> high
	// So we'll sort our layers, then ID them in order to make sure they're rendered
	// in the intended order.
	sort.Slice(m.ImageLayers, func(i, j int) bool {
		in, _ := strconv.ParseInt(m.ImageLayers[i].Name, 10, 64)
		jn, _ := strconv.ParseInt(m.ImageLayers[j].Name, 10, 64)
		return in < jn
	})
	sort.Slice(m.TileLayers, func(i, j int) bool {
		in, _ := strconv.ParseInt(m.TileLayers[i].Name, 10, 64)
		jn, _ := strconv.ParseInt(m.TileLayers[j].Name, 10, 64)
		return in < jn
	})
	for i, l := range m.ImageLayers {
		l.ID = uint(i + 1)
	}
	for i, l := range m.TileLayers {
		l.ID = uint(i + len(m.ImageLayers) + 1)
	}

	offset := uint(0)
	if len(m.Tilesets) > 0 { // TODO: this only supports one tileset
		offset += m.Tilesets[0].FirstGID
	}
	for _, tl := range m.TileLayers {
		ids := make([]uint, len(tl.decodedTiles))
		for i, j := range tl.decodedTiles {
			if j == 0 {
				ids[i] = 0
			} else {
				ids[i] = j + offset
			}
		}

		tdata, err := tl.Data.encodeCSV(m.Width, m.Height, ids)
		if err != nil {
			return err
		}
		tl.Data.RawData = tdata
	}

	return xml.NewEncoder(w).Encode(m)
}

// Decode an input TMX map XML
func Decode(r io.Reader) (*Map, error) {
	m := &Map{}
	if err := xml.NewDecoder(r).Decode(m); err != nil {
		return nil, err
	}

	if len(m.Tilesets) != 1 {
		return nil, fmt.Errorf("lib only supports 1 tileset")
	}

	m.nextID = uint(1)
	for _, ts := range m.Tilesets {
		ts.tileByID = map[uint]*Tile{}
		ts.tileBySrc = map[string]*Tile{}

		// I know we just checked if len tilesets != 1 but
		// in future we may support more
		for _, t := range ts.Tiles {
			if t.ID > m.nextID {
				m.nextID = t.ID + 1
			}

			ts.tileByID[t.ID+ts.FirstGID] = t
			ts.tileBySrc[t.Image.Source] = t
		}
	}

	for _, tl := range m.TileLayers {
		csvdata, err := tl.Data.decodeCSV()
		if err != nil {
			return nil, err
		}
		tl.decodedTiles = csvdata
	}

	return m, nil
}

//
func Open(fname string) (*Map, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	return Decode(f)
}

//
func (m *Map) WriteFile(fname string) error {
	buff := bytes.Buffer{}
	err := m.Encode(&buff)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fname, buff.Bytes(), 0644)
}
