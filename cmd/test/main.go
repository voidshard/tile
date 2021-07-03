package main

import (
	"os"

	"github.com/voidshard/tile"
)

func main() {
	m := tile.New(&tile.Config{
		TileWidth:  32,
		TileHeight: 32,
		MapWidth:   41,
		MapHeight:  41,
	})

	//	m.SetBackground("height.png")

	m.Set(5, 4, 0, "grass.png")
	m.Set(4, 4, 0, "grass.png")
	m.Set(4, 3, 0, "grass.png")
	m.Set(3, 4, 0, "grass.png")
	m.Set(5, 4, 1, "buck.png")

	o, err := tile.Open("/home/augustus/pkmn/tree.green.01.tmx")
	if err != nil {
		panic(err)
	}
	m.Add(10, 15, -1, o)
	m.Add(15, 25, 10, o)

	f, err := os.OpenFile("/tmp/map.tmx", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}

	err = m.Encode(f)
	if err != nil {
		panic(err)
	}
}
