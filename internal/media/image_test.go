package media

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/gen2brain/webp"
)

func TestReadImageAcceptsPNGJPEGAndWebP(t *testing.T) {
	source := testImage(96, 64)
	var jpegData, pngData, webpData bytes.Buffer
	if err := jpeg.Encode(&jpegData, source, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&pngData, source); err != nil {
		t.Fatal(err)
	}
	if err := webp.Encode(&webpData, source, webp.Options{Quality: 82, Method: 4}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		data []byte
	}{
		{name: "duck.png", data: pngData.Bytes()},
		{name: "duck.jpg", data: jpegData.Bytes()},
		{name: "duck.jpeg", data: jpegData.Bytes()},
		{name: "duck.webp", data: webpData.Bytes()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.name)
			if err := os.WriteFile(path, tt.data, 0600); err != nil {
				t.Fatal(err)
			}

			got, err := ReadImage(path)
			if err != nil {
				t.Fatalf("ReadImage() error = %v", err)
			}
			if got.Name != tt.name || got.MIMEType != "image/webp" {
				t.Fatalf("ReadImage() = {Name:%q MIMEType:%q}, want name %q and normalized MIME image/webp", got.Name, got.MIMEType, tt.name)
			}
			decoded, format, err := image.Decode(bytes.NewReader(got.Data))
			if err != nil {
				t.Fatalf("decode normalized image: %v", err)
			}
			if format != "webp" || decoded.Bounds().Dx() != 96 || decoded.Bounds().Dy() != 64 {
				t.Fatalf("normalized image = %s %v, want webp 96x64", format, decoded.Bounds())
			}
		})
	}
}

func TestReadImageRejectsExtensionSignatureMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duck.jpg")
	if err := os.WriteFile(path, []byte("\x89PNG\r\n\x1a\nimage"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadImage(path); err == nil {
		t.Fatal("ReadImage() accepted PNG bytes with a JPEG extension")
	}
}

func TestReadImageRejectsUnsupportedAndOversizeFiles(t *testing.T) {
	dir := t.TempDir()
	unsupported := filepath.Join(dir, "duck.gif")
	if err := os.WriteFile(unsupported, []byte("GIF89a"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadImage(unsupported); err == nil {
		t.Fatal("ReadImage() accepted GIF")
	}

	oversize := filepath.Join(dir, "large.png")
	if err := os.WriteFile(oversize, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(oversize, (10<<20)+1); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadImage(oversize); err == nil {
		t.Fatal("ReadImage() accepted a file larger than 10 MiB")
	}
}

func TestReadImageRejectsDirectoryAndUnreadablePath(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{dir, filepath.Join(dir, "missing.png")} {
		if _, err := ReadImage(path); err == nil {
			t.Errorf("ReadImage(%q) unexpectedly succeeded", path)
		}
	}
}

func TestReadImageNormalizesJPEGToWebPAndLeavesSourceUntouched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.jpg")
	var original bytes.Buffer
	if err := jpeg.Encode(&original, testImage(480, 320), &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	originalBytes := append([]byte(nil), original.Bytes()...)
	if err := os.WriteFile(path, originalBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	attachment, err := ReadImage(path)
	if err != nil {
		t.Fatalf("ReadImage() error = %v", err)
	}
	if attachment.MIMEType != "image/webp" {
		t.Fatalf("JPEG MIME type = %q, want image/webp", attachment.MIMEType)
	}
	if len(attachment.Data) < 12 || string(attachment.Data[:4]) != "RIFF" || string(attachment.Data[8:12]) != "WEBP" {
		t.Fatal("JPEG attachment was not encoded as WebP")
	}
	unchanged, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(unchanged, originalBytes) {
		t.Fatal("ReadImage() modified the source file")
	}
}

func TestReadImageResizesOversizedJPEGForDuckAI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large-photo.jpg")
	var original bytes.Buffer
	if err := jpeg.Encode(&original, testImage(1254, 1254), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	originalBytes := append([]byte(nil), original.Bytes()...)
	if err := os.WriteFile(path, originalBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	attachment, err := ReadImage(path)
	if err != nil {
		t.Fatalf("ReadImage() error = %v", err)
	}
	decoded, format, err := image.DecodeConfig(bytes.NewReader(attachment.Data))
	if err != nil {
		t.Fatalf("compressed WebP did not decode: %v", err)
	}
	if format != "webp" {
		t.Fatalf("resized format = %q, want webp", format)
	}
	if decoded.Width > maxDuckAIImageDimension || decoded.Height > maxDuckAIImageDimension {
		t.Fatalf("prepared image dimensions = %dx%d, want each side <= %d", decoded.Width, decoded.Height, maxDuckAIImageDimension)
	}
	unchanged, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(unchanged, originalBytes) {
		t.Fatal("ReadImage() modified the source file")
	}
}

func TestReadImageResizePreservesTranslucentPixelColors(t *testing.T) {
	// A uniform field of fully saturated red at 50% alpha. Resizing must not
	// apply alpha a second time, which would darken the color channels.
	const (
		width  = 1024
		height = 1024
	)
	source := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(source, source.Bounds(), &image.Uniform{color.NRGBA{R: 0xff, A: 0x80}}, image.Point{}, draw.Src)

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "translucent.png")
	if err := os.WriteFile(path, encoded.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	attachment, err := ReadImage(path)
	if err != nil {
		t.Fatalf("ReadImage() error = %v", err)
	}
	decoded, format, err := image.Decode(bytes.NewReader(attachment.Data))
	if err != nil {
		t.Fatalf("prepared image did not decode: %v", err)
	}
	if format != "webp" {
		t.Fatalf("prepared image format = %q, want webp", format)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(attachment.Data))
	if err != nil {
		t.Fatal(err)
	}
	if config.Width > maxDuckAIImageDimension || config.Height > maxDuckAIImageDimension {
		t.Fatalf("prepared image dimensions = %dx%d, want each side <= %d", config.Width, config.Height, maxDuckAIImageDimension)
	}

	center := decoded.Bounds().Min
	// Compare non-premultiplied channels: decoded.At().RGBA() would apply
	// alpha again and hide the very darkening this test guards against.
	got, ok := color.NRGBAModel.Convert(decoded.At(center.X, center.Y)).(color.NRGBA)
	if !ok {
		t.Fatalf("decoded pixel is not an NRGBA color")
	}
	if got.A != 0x80 {
		t.Fatalf("resized alpha = %#x, want 0x80", got.A)
	}
	// Tolerate rounding from bilinear resampling and the PNG round trip, but
	// catch double-alpha darkening, which would drop red to roughly half.
	if got.R < 0xf0 {
		t.Fatalf("resized red = %#x, want ~0xff; alpha appears applied twice", got.R)
	}
	if got.G > 0x10 || got.B > 0x10 {
		t.Fatalf("resized green/blue = %#x/%#x, want ~0x00; the opaque red field leaked", got.G, got.B)
	}
}

func TestReadImageNormalizesJPEGWithEXIFMetadataToWebP(t *testing.T) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, testImage(128, 96), &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}

	// JPEG APP1 segment with the EXIF identifier.
	original := append([]byte(nil), encoded.Bytes()[:2]...)
	original = append(original, []byte{0xff, 0xe1, 0x00, 0x08, 'E', 'x', 'i', 'f', 0x00, 0x00}...)
	original = append(original, encoded.Bytes()[2:]...)
	path := filepath.Join(t.TempDir(), "photo.jpg")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}

	attachment, err := ReadImage(path)
	if err != nil {
		t.Fatalf("ReadImage() error = %v", err)
	}
	if attachment.MIMEType != "image/webp" {
		t.Fatalf("normalized MIME type = %q, want image/webp", attachment.MIMEType)
	}
	if _, format, err := image.Decode(bytes.NewReader(attachment.Data)); err != nil || format != "webp" {
		t.Fatalf("normalized image format = %q, err=%v; want webp", format, err)
	}
}

func TestReadImageAppliesJPEGEXIFOrientationBeforeWebPEncoding(t *testing.T) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, testImage(32, 64), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	data := jpegWithOrientation(encoded.Bytes(), 6)
	path := filepath.Join(t.TempDir(), "portrait.jpg")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	attachment, err := ReadImage(path)
	if err != nil {
		t.Fatalf("ReadImage() error = %v", err)
	}
	decoded, format, err := image.DecodeConfig(bytes.NewReader(attachment.Data))
	if err != nil {
		t.Fatalf("decode normalized image: %v", err)
	}
	if format != "webp" || decoded.Width != 64 || decoded.Height != 32 {
		t.Fatalf("normalized image = %s %dx%d, want webp 64x32 after EXIF orientation", format, decoded.Width, decoded.Height)
	}
}

func TestReadImageNormalizesOversizedJPEGWithEXIFToWebP(t *testing.T) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, testImage(1254, 900), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), encoded.Bytes()[:2]...)
	original = append(original, []byte{0xff, 0xe1, 0x00, 0x08, 'E', 'x', 'i', 'f', 0x00, 0x00}...)
	original = append(original, encoded.Bytes()[2:]...)
	path := filepath.Join(t.TempDir(), "oriented-photo.jpg")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}

	attachment, err := ReadImage(path)
	if err != nil {
		t.Fatalf("ReadImage() error = %v", err)
	}
	if attachment.MIMEType != "image/webp" {
		t.Fatalf("normalized MIME type = %q, want image/webp", attachment.MIMEType)
	}
	decoded, format, err := image.DecodeConfig(bytes.NewReader(attachment.Data))
	if err != nil {
		t.Fatalf("resized WebP did not decode: %v", err)
	}
	if format != "webp" {
		t.Fatalf("resized image format = %q, want webp", format)
	}
	if decoded.Width > maxDuckAIImageDimension || decoded.Height > maxDuckAIImageDimension {
		t.Fatalf("resized image dimensions = %dx%d, want each side <= %d", decoded.Width, decoded.Height, maxDuckAIImageDimension)
	}
}

func TestReadImageNormalizesPNGToWebP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	source := testPatternImage(512, 384)
	var original bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&original, source); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, original.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	attachment, err := ReadImage(path)
	if err != nil {
		t.Fatalf("ReadImage() error = %v", err)
	}
	if attachment.MIMEType != "image/webp" {
		t.Fatalf("PNG MIME type = %q, want image/webp", attachment.MIMEType)
	}
	decoded, format, err := image.Decode(bytes.NewReader(attachment.Data))
	if err != nil {
		t.Fatalf("normalized WebP did not decode: %v", err)
	}
	if format != "webp" || decoded.Bounds() != source.Bounds() {
		t.Fatalf("normalized PNG output = %s %v, want webp %v", format, decoded.Bounds(), source.Bounds())
	}
}

func testImage(width, height int) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, width, height))
	state := uint32(1)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			state = state*1664525 + 1013904223
			result.SetRGBA(x, y, color.RGBA{R: byte(state >> 24), G: byte(state >> 16), B: byte(state >> 8), A: 255})
		}
	}
	return result
}

func testPatternImage(width, height int) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			result.SetRGBA(x, y, color.RGBA{R: byte((x/16)%16) * 16, G: byte((y/16)%16) * 16, B: byte(((x+y)/16)%16) * 16, A: 255})
		}
	}
	return result
}

func jpegWithOrientation(data []byte, orientation uint16) []byte {
	exif := []byte{
		'E', 'x', 'i', 'f', 0, 0,
		'I', 'I', 42, 0, 8, 0, 0, 0,
		1, 0,
		0x12, 0x01, 3, 0, 1, 0, 0, 0,
		byte(orientation), byte(orientation >> 8), 0, 0,
		0, 0, 0, 0,
	}
	segmentLength := len(exif) + 2
	segment := []byte{0xff, 0xe1, byte(segmentLength >> 8), byte(segmentLength)}
	segment = append(segment, exif...)
	result := append([]byte(nil), data[:2]...)
	result = append(result, segment...)
	return append(result, data[2:]...)
}
