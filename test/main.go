package main

import (
	"fmt"

	"github.com/voidshard/tile"
)

const grass = "grass.png"
const mushroom = "mushroom.png"

func main() {
	/*Designed to be run from the repo root
	 */

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
	m.Set(1, 2, 1, mushroom)

	tree, err := tile.Open("test/tree.large.01.tmx")
	if err != nil {
		panic(err)
	}

	canFit := m.Fits(3, 3, 2, tree)
	if !canFit {
		panic(fmt.Sprintf("we should be able to fit this"))
	}
	fmt.Println("fits", canFit)
	m.Add(3, 3, 2, tree)

	err = m.WriteFile("test/one.tmx")
	if err != nil {
		panic(err)
	}

	canFit = m.Fits(3, 3, 2, tree)
	if canFit {
		panic(fmt.Sprintf("this is copying directly over the first tree"))
	}
}
