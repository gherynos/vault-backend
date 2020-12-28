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

func Decode(data string) (bt []byte, err error) {

	var decoder io.ReadCloser
	if decoder, err = zlib.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))); err != nil {

		return
	}

	defer decoder.Close()
	bt, err = ioutil.ReadAll(decoder)
	if err != nil && err != io.ErrUnexpectedEOF {

		return

	} else {

		err = nil
	}

	return
}
