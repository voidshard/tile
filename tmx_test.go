package tile

import (
	"github.com/stretchr/testify/assert"

	"bytes"
	"testing"
)

func TestDecode(t *testing.T) {
	m, err := Decode(bytes.NewBuffer([]byte(csvdata)))

	assert.Nil(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, 1, len(m.Tilesets))
	assert.Equal(t, 1, len(m.TileLayers))
	assert.Equal(t, m.Width, 10)
	assert.Equal(t, m.Height, 10)
	assert.Equal(t, m.TileWidth, 32)
	assert.Equal(t, m.TileHeight, 32)
	assert.Equal(t, len(m.TileLayers[0].decodedTiles), 1024)
}

func TestEncode(t *testing.T) {
	m, err := Decode(bytes.NewBuffer([]byte(csvdata)))

	assert.Nil(t, err)
	if err != nil {
		return
	}

	buf := bytes.Buffer{}
	err = m.Encode(&buf)

	assert.Nil(t, err)
	assert.Equal(t, csvReEncoded, string(buf.Bytes()))
}
