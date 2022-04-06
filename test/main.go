package main

import (
	"fmt"

	"github.com/voidshard/tile"
)

const grass = "grass.png"
const mushroom = "mushroom.png"
const treepiece = "tree.large.01.1.3.0.png"

func main() {
	/*Designed to be run from the repo root
	 */
	err := testOne()
	if err != nil {
		fmt.Println("test one:", err)
	}

	err = testTwo()
	if err != nil {
		fmt.Println("test two:", err)
	}

	err = testThree()
	if err != nil {
		fmt.Println("test three:", err)
	}
}

func testThree() error {
	// map should be same as map two - this proves we can re-use an
	// existing sqlite db file.
	// Not sure why we shouldn't be able to but .. eh.
	inf, err := tile.NewInfiniteMap()
	if err != nil {
		return err
	}

	err = decorate(inf)
	if err != nil {
		return err
	}

	newinf, err := tile.OpenInfiniteMap(inf.Filename())
	if err != nil {
		return err
	}

	m, err := newinf.Map(32, 32, 0, 0, 10, 10)
	if err != nil {
		return err
	}

	err = m.WriteFile("test/three.tmx")
	if err != nil {
		return err
	}

	return nil
}

func testTwo() error {
	// the same as testOne but using an infinite map instead
	inf, err := tile.NewInfiniteMap()
	if err != nil {
		return err
	}

	err = decorate(inf)
	if err != nil {
		return err
	}

	m, err := inf.Map(32, 32, 0, 0, 10, 10)
	if err != nil {
		return err
	}

	err = m.WriteFile("test/two.tmx")
	if err != nil {
		return err
	}

	return nil
}

func decorate(m tile.Tileable) error {
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			m.Set(x, y, 0, grass)
		}
	}
	err := m.Set(1, 2, 1, mushroom)
	if err != nil {
		return err
	}

	mushsrc, err := m.At(1, 2, 1)
	if err != nil {
		return err
	}
	if mushsrc != mushroom {
		return fmt.Errorf("failed to read src back, got %s expected %s", mushsrc, mushroom)
	}

	fprops := tile.NewProperties()
	fprops.SetString("hello", "world")
	fprops.SetInt("one", 1)
	fprops.SetBool("bool", false)
	err = m.SetProperties(mushroom, fprops)
	if err != nil {
		return err
	}

	err = m.SetProperties(treepiece, fprops) // belongs in tree.large.01
	if err != nil {
		return err
	}

	tree, err := tile.Open("test/tree.large.01.tmx")
	if err != nil {
		return err
	}

	canFit, err := m.Fits(3, 3, 2, tree)
	if err != nil {
		return err
	}
	if !canFit {
		return fmt.Errorf("we should be able to fit this")
	}
	err = m.Add(3, 3, 2, tree)
	if err != nil {
		return err
	}

	canFit, err = m.Fits(3, 3, 2, tree)
	if err != nil {
		return err
	}
	if canFit {
		return fmt.Errorf("this is copying directly over the first tree")
	}

	return nil
}

func testOne() error {
	// make a map, decorate it, write it out
	m := tile.New(&tile.Config{
		MapWidth:   10,
		MapHeight:  10,
		TileWidth:  32,
		TileHeight: 32,
	})

	err := decorate(m)
	if err != nil {
		return err
	}

	err = m.WriteFile("test/one.tmx")
	if err != nil {
		return err
	}

	return nil
}
