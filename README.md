## Tile

Tiled TMX map compositing.


### Tile Objects

This library leverages the TMX format to think about some maps as 'tile objects' ('tobs') that can be copied onto larger maps.

In this way you can have TMX maps that contain trees, houses, fences or whatever that span multiple z-levels & easily copy them onto a larger map starting at some z-level. We can thus place complicated objects easily in much the same way we might place sprites (and have map layers automatically created / added to as required).

This allows us to build large Tiled maps by placing 'bridge.01' 'tree.14' without needing to worry about placing each tile or their layers / offsets.


### Tob tool 

A small binary `tob` is included that can efficiently cut tob TMX maps out of a large image (eg. a sprite sheet).

For example, this writes out a tree tob that is
- cut out of a large image 'spritesheet.png'
- 4 z-levels high (spread over 4 TileLayers)
- 4 tiles high (4 rows of tiles)
- 2 tiles wide (2 columns of tiles)
- includes metadata 'biome', 'impassable', 'object'

```bash
> tob -i ./src/spritesheet.png -n tree.large.01 -p biome=forest -p impassable=true -p object=tree -z 1 -z 2 -z 3 64 9408 2t 4t 
```

Pass input image
`-i ./src/spritesheet.png`

Specify output image & TMX prefix
`-n tree.large.01`

Set some metadata (supports string, int, bool) - these are saved as map properties
`-p biome=forest -p impassable=true -p object=tree`

At levels 1,2 & 3 add a z-level. Levels are measured in tiles starting from the bottom
`-z 1 -z 2 -z 3`
This means that the bottom row of tree tiles will be saved on layer-0, the next row on layer-1 etc. It so happens here we climb a z-level on each row (because the tree is near vertical) but we need not do this. If you want the higher rows to be on lower TileLayers then you can pass `--invert`

Specify where in the input image to look
`64 9408 2t 4t`
This is in the format 'x0 y0 x1 y1' and gives the bounds of the input area (similar to Go's Image package). The thing to note here is the short hand '2t' and '4t' which says '2 tiles wide' and '4 tiles high'. You can also give absolute pixel values if you want to. By default the tile width & height is 32. You can set this with `--tile-width` and `--tile-height`

```bash
> ls -1 tree.large.01.* 
tree.large.01.0.0.30.png
tree.large.01.0.1.20.png
tree.large.01.0.2.10.png
tree.large.01.0.3.0.png
tree.large.01.1.0.30.png
tree.large.01.1.1.20.png
tree.large.01.1.2.10.png
tree.large.01.1.3.0.png
tree.large.01.tmx
```
This creates images with the names `<prefix>.<x>.<y>.<z>.png` and a matching .tmx file. You'll note that the z levels are 0, 10, 20, 30 rather than 0-3. By default we multiply the z layer by 10 in order to leave room for say, hand made additions or any other finishing touches (see `--mult`).

Since the .tmx file includes it's own tileset that references the images it needs we can directly open this with the Tiled editor to check it out.

