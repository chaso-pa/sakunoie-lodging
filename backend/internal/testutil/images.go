package testutil

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// GPS position written into the EXIF of JPEGWithGPSExif: 35°39'31"N
// 139°44'43"E.
var exifGPSRationals = [2][3][2]uint32{
	{{35, 1}, {39, 1}, {31, 1}},
	{{139, 1}, {44, 1}, {43, 1}},
}

// JPEGWithGPSExif returns a width x height JPEG whose EXIF block carries a
// GPS position and Orientation=6 (rotate 90° clockwise to display).
func JPEGWithGPSExif(t testing.TB, width, height int) []byte {
	t.Helper()
	plain := PlainJPEG(t, width, height)

	// TIFF, little endian. IFD0 at 8 holds Orientation and the pointer to
	// the GPS IFD at 38, whose two rational triples sit at 92 and 116.
	le := binary.LittleEndian
	tiff := make([]byte, 140)
	copy(tiff, "II*\x00")
	le.PutUint32(tiff[4:], 8)
	entry := func(off int, tag, typ uint16, count, value uint32) {
		le.PutUint16(tiff[off:], tag)
		le.PutUint16(tiff[off+2:], typ)
		le.PutUint32(tiff[off+4:], count)
		le.PutUint32(tiff[off+8:], value)
	}
	le.PutUint16(tiff[8:], 2)
	entry(10, 0x0112, 3, 1, 6)  // Orientation
	entry(22, 0x8825, 4, 1, 38) // GPSInfo IFD pointer
	le.PutUint16(tiff[38:], 4)
	entry(40, 0x0001, 2, 2, uint32('N')) // GPSLatitudeRef
	entry(52, 0x0002, 5, 3, 92)          // GPSLatitude
	entry(64, 0x0003, 2, 2, uint32('E')) // GPSLongitudeRef
	entry(76, 0x0004, 5, 3, 116)         // GPSLongitude
	for i, triple := range exifGPSRationals {
		for j, rational := range triple {
			off := 92 + i*24 + j*8
			le.PutUint32(tiff[off:], rational[0])
			le.PutUint32(tiff[off+4:], rational[1])
		}
	}

	payload := append([]byte("Exif\x00\x00"), tiff...)
	var out bytes.Buffer
	out.Write(plain[:2]) // SOI
	out.Write([]byte{0xFF, 0xE1})
	binary.Write(&out, binary.BigEndian, uint16(len(payload)+2))
	out.Write(payload)
	out.Write(plain[2:])

	if !HasExif(out.Bytes()) {
		t.Fatal("test JPEG was built without EXIF")
	}
	return out.Bytes()
}

// PlainJPEG returns a JPEG whose left half is red and right half blue, so
// rotation can be told from the pixels.
func PlainJPEG(t testing.TB, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, halves(width, height), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// PlainPNG returns a PNG with the same picture as PlainJPEG.
func PlainPNG(t testing.TB, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, halves(width, height)); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func halves(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.RGBA{R: 220, A: 255}
			if x >= width/2 {
				c = color.RGBA{B: 220, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// JPEGSegments returns the marker bytes of the segments before the image
// data (0xE0 for APP0, 0xE1 for APP1/EXIF, 0xDB for DQT and so on).
func JPEGSegments(data []byte) []byte {
	var markers []byte
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil
	}
	for i := 2; i+4 <= len(data) && data[i] == 0xFF; {
		marker := data[i+1]
		markers = append(markers, marker)
		if marker == 0xDA { // start of scan: image data follows
			break
		}
		i += 2 + int(binary.BigEndian.Uint16(data[i+2:]))
	}
	return markers
}

// HasExif reports whether data carries any EXIF: an APP1 segment, the Exif
// header, or the GPS rationals JPEGWithGPSExif writes.
func HasExif(data []byte) bool {
	for _, marker := range JPEGSegments(data) {
		if marker == 0xE1 {
			return true
		}
	}
	gps := make([]byte, 0, 24)
	for _, rational := range exifGPSRationals[0] {
		gps = binary.LittleEndian.AppendUint32(gps, rational[0])
		gps = binary.LittleEndian.AppendUint32(gps, rational[1])
	}
	return bytes.Contains(data, []byte("Exif")) || bytes.Contains(data, gps)
}
