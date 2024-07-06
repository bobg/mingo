// I am indebted to https://github.com/alexdyukov for this code
// and the attendant bug report.
// See https://github.com/bobg/mingo/issues/14

package main

import (
	"bytes"
	"fmt"
	"io"
)

func main() {
	fmt.Println(len(customReadAll(io.NopCloser(&bytes.Buffer{}))))
}

func customReadAll(reader io.Reader) []byte {
	buf := make([]byte, 100)
	retVal := []byte{}
	for {
		n, err := reader.Read(buf)

		retVal = append(retVal, buf...)
		buf = buf[:0]

		if n < 100 || err != nil {
			return retVal
		}
	}

	return nil
}
