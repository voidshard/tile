package main

import (
	"fmt"

	"github.com/voidshard/tile"
)

const grass = "grass.summer.01.0.0.0.png"

func main() {
	m := tile.New(&tile.Config{
		MapWidth:   10,
		MapHeight:  10,
		TileWidth:  32,
		TileHeight: 32,
	})

	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			m.Set(x, y, 0, grass)
		}
	}

	tree, err := tile.Open("tree.large.01.tmx")
	if err != nil {
		panic(err)
	}

	canFit := m.Fits(3, 3, 1, tree)
	if !canFit {
		panic(fmt.Sprintf("we should be able to fit this"))
	}
	fmt.Println("fits", canFit)
	m.Add(3, 3, 1, tree)

	err = m.WriteFile("one.tmx")
	if err != nil {
		panic(err)
	}

	canFit = m.Fits(3, 3, 1, tree)
	if canFit {
		panic(fmt.Sprintf("this is copying directly over the first tree"))
	}
}
