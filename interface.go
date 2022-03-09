package tile

// Tileable represents something we can tile
type Tileable interface {
	// Set a single tile (given src image) at x,y,z
	Set(x, y, z int, src string) error

	// Add an object `o` beginning at x,y,z
	// Any set properties on tiles in `o` will be merged
	Add(x, y, z int, o *Map) error

	// Fits returns if placing an object `o` beginning at x,y,z
	// would cause us to overwrite any currently set tile
	Fits(x, y, z int, o *Map) (bool, error)

	// Properties gets properties (if set) on the given src
	Properties(src string) (*Properties, error)

	// SetProperties sets properties on the given src
	SetProperties(src string, props *Properties) error
}
