package tile

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const (
	sqlUpdateTiles = `INSERT INTO tiles (id, x, y, z, src) VALUES (:id, :x, :y, :z, :src) ON CONFLICT (id) DO UPDATE SET src=EXCLUDED.src;`
	sqlGetProps    = `SELECT src,data FROM properties WHERE `
	sqlUpdateProps = `INSERT INTO properties (src, data) VALUES (:src, :data) ON CONFLICT (src) DO UPDATE SET data=EXCLUDED.data;`
)

// namedQuery allows us to use either a transaction.NamedQuery or DB.NamedQuery
// in our sub functions.
// Tl;dr it's helpful for using the same code in & out of transactions.
type namedQuery func(string, interface{}) (*sqlx.Rows, error)

// NewInfiniteMap creates an 'infinite' version of a 'tileable' map.
// This creates a random name for the database & stores it in the os tempdir.
func NewInfiniteMap() (*InfiniteMap, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	fname := filepath.Join(os.TempDir(), fmt.Sprintf("infmap.%d.sqlite", rng.Intn(1000000)))
	return OpenInfiniteMap(fname)
}

// OpenInfiniteMap given it's filename (database file) on disk.
// Will create if it doesn't exist.
func OpenInfiniteMap(fname string) (*InfiniteMap, error) {
	db, err := sqlx.Open("sqlite3", fname)
	if err != nil {
		return nil, err
	}

	inf := &InfiniteMap{db: db, filename: fname}
	return inf, inf.init()
}

// InfiniteMap holds all the same data as a 'Map' (an in memory .TMX map)
// but regluarly flushed to disk (in another format) so we can hold truly
// massive maps.
//
// We can then use this to write out any number of .tmx maps of practical
// sizes for use in other systems.
type InfiniteMap struct {
	filename string
	db       *sqlx.DB
}

// Filename returns the path to the infinite map data on disk
func (i *InfiniteMap) Filename() string {
	return i.filename
}

// Map returns a (Tile)Map with all tiles from the infinite map in the rectangle (x0,y0,x1,y1).
func (i *InfiniteMap) Map(tilewidth, tileheight uint, x0, y0, x1, y1 int) (*Map, error) {
	if x1 <= x0 || y1 <= y0 {
		return nil, fmt.Errorf("requested map dimensions invalid, unable to render map")
	}

	tmap := &Map{
		Orientation:    "orthogonal",
		Width:          x1 - x0,
		Height:         y1 - y0,
		TileWidth:      int(tilewidth),
		TileHeight:     int(tileheight),
		Tilesets:       []*Tileset{newTileset("default", 1)},
		RootProperties: []*Property{},
		TileLayers:     []*TileLayer{},
		ImageLayers:    []*ImageLayer{},
		nextID:         1,
	}

	rows, err := i.db.NamedQuery(
		"SELECT x,y,z,src FROM tiles WHERE x>=:x0 AND x<:x1 AND y>=:y0 AND y<:y1;",
		map[string]interface{}{
			"x0": x0, "x1": x1,
			"y0": y0, "y1": y1,
		},
	)
	if err != nil {
		return nil, err
	}

	srcs := []string{}
	tile := dbTile{}
	for rows.Next() {
		rows.StructScan(&tile)
		srcs = append(srcs, tile.Src)
		tmap.Set(tile.X, tile.Y, tile.Z, tile.Src)
	}

	srcProps, err := i.properties(i.db.NamedQuery, srcs...)
	if err != nil {
		return nil, err
	}

	for src, props := range srcProps {
		tmap.SetProperties(src, props)
	}

	return tmap, nil
}

// At returns the tile that exists at the given location (or "" if unset)
func (i *InfiniteMap) At(x, y, z int) (string, error) {
	rows, err := i.db.NamedQuery(
		"SELECT x,y,z,src FROM tiles WHERE x=:x0 AND y=:y0 AND z=:z0 LIMIT 1;",
		map[string]interface{}{
			"x0": x,
			"y0": y,
			"z0": z,
		},
	)
	if err != nil {
		return "", err
	}

	tile := dbTile{}
	for rows.Next() { // there's at most one due to LIMIT 1
		rows.StructScan(&tile)
	}

	return tile.Src, nil
}

// Set the given image src at (x,y,z)
func (i *InfiniteMap) Set(x, y, z int, src string) error {
	_, err := i.db.NamedExec(sqlUpdateTiles, newDBTile(x, y, z, src))
	return err
}

// Add the given tile object map `0` beginning at (x,y,z)
func (i *InfiniteMap) Add(x, y, zoffset int, o *Map) error {
	updateTiles := []dbTile{}

	srcsToUpdate := []string{}
	propsCurrent := map[string]*Properties{}

	ts := o.Tilesets[0]
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

			updateTiles = append(updateTiles, newDBTile(tx+x, ty+y, int(z)+zoffset, src))
			oprops, _ := o.Properties(src)
			propsCurrent[src] = oprops
			srcsToUpdate = append(srcsToUpdate, src)
		}
	}

	// insert tiles
	_, err := i.db.NamedExec(sqlUpdateTiles, updateTiles)
	if err != nil {
		return err
	}

	// update properties in a transaction
	txn, err := i.db.Beginx()
	if err != nil {
		return err
	}

	existingProps, err := i.properties(txn.NamedQuery, srcsToUpdate...)
	if err != nil {
		txn.Rollback()
		return err
	}

	propStructs := []dbProp{}
	for src, now := range propsCurrent {
		saved, _ := existingProps[src]
		if saved == nil {
			saved = NewProperties()
		}

		propStructs = append(propStructs, newDBProp(src, saved.Merge(now)))
	}

	_, err = txn.NamedExec(sqlUpdateProps, propStructs)
	if err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}

// Fits returns if writing the given tilemap `o` starting at (x,y,z) would require
// overwriting an already set tile.
// Nb. we do not check nil (empty) tiles in the given object but rather if there
// are tiles set in the rectangle described starting from (x,y,z) and adding
// the object width, height and it's highest z-layer.
func (i *InfiniteMap) Fits(x, y, z int, o *Map) (bool, error) {
	highest := 0
	lvls := o.ZLevels()
	if len(lvls) > 0 {
		highest = lvls[len(lvls)-1]
	}

	rows, err := i.db.NamedQuery(
		"SELECT count(*) as num FROM tiles WHERE x>=:x0 AND x<:x1 AND y>=:y0 AND y<:y1 AND z>=:z0 AND z<:z1;",
		map[string]interface{}{
			"x0": x, "x1": x + o.Width,
			"y0": y, "y1": y + o.Height,
			"z0": z, "z1": z + highest + 1, // since `highest` is the z-layer (eg, 0 means "the first layer")
		},
	)
	if err != nil {
		return false, err
	}

	var num int64
	for rows.Next() { // should only be one row
		rows.Scan(&num)
	}

	return num == 0, nil
}

// properties returns set properties by their src name
func (i *InfiniteMap) properties(do namedQuery, in ...string) (map[string]*Properties, error) {
	args := map[string]interface{}{}
	or := []string{}

	for i, src := range in {
		name := fmt.Sprintf("prop_%d", i)

		args[name] = src
		or = append(or, fmt.Sprintf("src=:%s", name))
	}

	qstr := fmt.Sprintf("%s %s LIMIT %d;", sqlGetProps, strings.Join(or, " OR "), len(in))

	rows, err := do(qstr, args)
	if err != nil {
		return nil, err
	}

	result := map[string]*Properties{}

	r := dbProp{}
	dblock := struct {
		I map[string]int
		S map[string]string
		B map[string]bool
	}{}
	for rows.Next() {
		err = rows.StructScan(&r)
		if err != nil {
			return nil, err
		}

		// explicit reset to results don't bleed together
		dblock.I = nil
		dblock.S = nil
		dblock.B = nil

		err = json.Unmarshal([]byte(r.Data), &dblock)
		if err != nil {
			return nil, err
		}

		result[r.Src] = &Properties{
			ints:    dblock.I,
			strings: dblock.S,
			bools:   dblock.B,
		}
	}

	return result, nil
}

// Properties returns properties for a given src
// Asking for "" (the empty tile) always returns nil
// Otherwise if no properties are set an empty properties will be returned.
func (i *InfiniteMap) Properties(src string) (*Properties, error) {
	if src == "" {
		return nil, nil
	}

	result, err := i.properties(i.db.NamedQuery, src)
	if err != nil {
		return nil, err
	}

	props, _ := result[src]
	if props == nil {
		return NewProperties(), nil
	}

	return props, nil
}

// SetProperties for the given src. This doesn't do an update / merge just overwrites.
func (i *InfiniteMap) SetProperties(src string, props *Properties) error {
	_, err := i.db.NamedExec(sqlUpdateProps, newDBProp(src, props))
	return err
}

// init creates some DB tables for us if they don't exist
func (i *InfiniteMap) init() error {
	createTiles := `CREATE TABLE IF NOT EXISTS tiles(
		id TEXT PRIMARY KEY,
		x INTEGER NOT NULL,
		y INTEGER NOT NULL,
		z INTEGER NOT NULL,
		src TEXT NOT NULL
	    );`
	_, err := i.db.Exec(createTiles)
	if err != nil {
		return err
	}

	createProps := `CREATE TABLE IF NOT EXISTS properties(
		src TEXT PRIMARY KEY,
		data TEXT
	    );`

	_, err = i.db.Exec(createProps)
	return err
}

// dbTile object encodes a single tile.
// The ID here is used to insert/update on a unique tile by it's (x,y,z)
// with a more straight forward query.
type dbTile struct {
	ID  string `db:"id"`
	X   int    `db:"x"`
	Y   int    `db:"y"`
	Z   int    `db:"z"`
	Src string `db:"src"`
}

// newDBTile crafts a dbTile struct given it's inputs
func newDBTile(x, y, z int, src string) dbTile {
	return dbTile{ID: fmt.Sprintf("%d-%d-%d", x, y, z), X: x, Y: y, Z: z, Src: src}
}

// dbProp object encodes properties for a single src.
type dbProp struct {
	Src  string `db:"src"`
	Data string `db:"data"`
}

// newDBProp crats a dbProp struct given it's inputs.
// Properties are encoded into JSON.
func newDBProp(src string, props *Properties) dbProp {
	dblock := struct {
		I map[string]int
		S map[string]string
		B map[string]bool
	}{
		props.ints,
		props.strings,
		props.bools,
	}

	databytes, _ := json.Marshal(dblock)

	return dbProp{Src: src, Data: string(databytes)}
}
