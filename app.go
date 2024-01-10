package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/adrium/goheif"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type writerSkipper struct {
	w           io.Writer
	bytesToSkip int
}

func (w *writerSkipper) Write(data []byte) (int, error) {
	if w.bytesToSkip <= 0 {
		return w.w.Write(data)
	}

	if dataLen := len(data); dataLen < w.bytesToSkip {
		w.bytesToSkip -= dataLen
		return dataLen, nil
	}

	if n, err := w.w.Write(data[w.bytesToSkip:]); err == nil {
		n += w.bytesToSkip
		w.bytesToSkip = 0
		return n, nil
	} else {
		return n, err
	}
}

func newWriterExif(w io.Writer, exif []byte) (io.Writer, error) {
	writer := &writerSkipper{w, 2}
	soi := []byte{0xff, 0xd8}
	if _, err := w.Write(soi); err != nil {
		return nil, err
	}

	if exif != nil {
		app1Marker := 0xe1
		markerlen := 2 + len(exif)
		marker := []byte{0xff, uint8(app1Marker), uint8(markerlen >> 8), uint8(markerlen & 0xff)}
		if _, err := w.Write(marker); err != nil {
			return nil, err
		}

		if _, err := w.Write(exif); err != nil {
			return nil, err
		}
	}

	return writer, nil
}

func heicToJpg(file *zip.File) (bytes.Buffer, error) {
	ioReader, _ := file.Open()
	buff := bytes.NewBuffer([]byte{})
	_, err := io.Copy(buff, ioReader)
	check(err)

	reader := bytes.NewReader(buff.Bytes())

	exif, err := goheif.ExtractExif(reader)
	if err != nil {
		panic(err)
	}

	img, err := goheif.Decode(reader)
	if err != nil {
		panic(err)
	}

	var b bytes.Buffer
	f := io.Writer(&b)
	w, err := newWriterExif(f, exif)

	check(err)
	err = jpeg.Encode(w, img, nil)

	return b, nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Missing parameter, provide file name!")
		return
	}
	outfile := os.Args[2]
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("Can't read file:", os.Args[1])
		panic(err)
	}

	mimeType := http.DetectContentType(data)
	if mimeType != "application/zip" {
		panic("unsupport file format")
	}

	reader := bytes.NewReader(data)
	zipReader, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		panic(err)
	}
	if len(zipReader.File) == 0 {
		panic("Empty zip")
	}

	for _, file := range zipReader.File {
		if strings.Contains(file.Name, ".HEIC") {
			b, err := heicToJpg(file)
			if err != nil {
				panic(err)
			}

			writeErr := os.WriteFile(outfile, b.Bytes(), 0644)
			if writeErr != nil {
				panic(writeErr)
			}
		}
	}
}
