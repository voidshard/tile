package tile

import ()

// Config includes settings for a TileMap
type Config struct {
	// in tiles
	MapHeight uint
	MapWidth  uint

	// in pixels
	TileWidth  uint
	TileHeight uint
}

// DefaultConfig returns a map config with default settings.
func DefaultConfig() *Config {
	return &Config{
		TileWidth:  32,
		TileHeight: 32,
		MapWidth:   100,
		MapHeight:  100,
	}
}
