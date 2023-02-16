package vault

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {

	data := []byte("This is a test string...")

	enc, err := Encode(data)

	assert.Nil(t, err)

	dec, err2 := Decode(enc)

	assert.Nil(t, err2)

	assert.Equal(t, data, dec)
}

func TestEncodeDecodeBin(t *testing.T) {

	data := make([]byte, 1024)
	rand.Read(data)

	enc, err := Encode(data)

	assert.Nil(t, err)

	dec, err2 := Decode(enc)

	assert.Nil(t, err2)

	assert.Equal(t, data, dec)
}
