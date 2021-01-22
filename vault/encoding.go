package vault

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"strings"
)

// Encode compresses data using ZLIB and returns its BASE64 representation.
func Encode(data []byte) (str string, err error) {

	var buf bytes.Buffer

	encoder := zlib.NewWriter(base64.NewEncoder(base64.StdEncoding, &buf))

	var w int
	defer encoder.Close()
	if w, err = encoder.Write(data); err != nil {

		return

	} else if w != len(data) {

		err = errors.New("not all the bytes were encoded")
		return
	}

	if err = encoder.Flush(); err != nil {

		return
	}

	str = buf.String()
	return
}

// Decode converts and decompresses the BASE64-encoded data, returning the original byte array.
func Decode(data string) (bt []byte, err error) {

	var decoder io.ReadCloser
	if decoder, err = zlib.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))); err != nil {

		return
	}

	defer decoder.Close()
	bt, err = ioutil.ReadAll(decoder)
	if err != nil && err != io.ErrUnexpectedEOF {

		return
	}

	err = nil
	return
}
