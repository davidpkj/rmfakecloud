package exporter

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	v6 "github.com/ddvk/reader/v6"
	"github.com/go-pdf/fpdf"
	rm2pdf "github.com/poundifdef/go-remarkable2pdf"
	"github.com/sirupsen/logrus"
)

// RenderPoundifdef caligraphy pen is nice
func RenderPoundifdef(input, output string) (io.ReadCloser, error) {
	reader, err := zip.OpenReader(input)
	if err != nil {
		return nil, fmt.Errorf("can't open file %w", err)
	}
	defer reader.Close()

	writer, err := os.Create(output)
	if err != nil {
		return nil, fmt.Errorf("can't create outputfile %w", err)
	}

	err = rm2pdf.RenderRmNotebookFromZip(&reader.Reader, writer)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("can't render file %w", err)
	}

	_, err = writer.Seek(0, 0)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("can't rewind file %w", err)
	}

	return writer, nil
}

// RenderRmapi renders with rmapi
func RenderRmapi(a *MyArchive, output io.Writer) error {
	pdfgen := PdfGenerator{}
	options := PdfGeneratorOptions{
		AllPages: true,
	}
	return pdfgen.Generate(a, output, options)
}

func parseSceneFile(file io.ReadCloser) (pdf *fpdf.Fpdf, err error) {
	headerLength := 0x2b
	buffer := make([]byte, headerLength)
	_, err = io.ReadFull(file, buffer)
	if err != nil {
		return nil, err
	}

	sceneParser := v6.SceneReader{}
	scene, err := sceneParser.ExtractScene(file)
	if err != nil {
		return nil, err
	}

	pdf = fpdf.NewCustom(&fpdf.InitType{
		UnitStr: "pt",
		Size:    fpdf.SizeType{Wd: DeviceWidth, Ht: DeviceHeight},
	})
	pdf.SetLineCapStyle("round")
	pdf.SetLineJoinStyle("round")

	// fmt.Printf("Number of Layer: %d\n", len(scene.Layers))
	for _, layer := range scene.Layers {
		for _, line := range layer.Lines {
			var lastPoint *v6.PenPoint
			for _, point := range line.Line.Value.Points {
				if lastPoint != nil {
					// logrus.Debug(point.String())

					w := float64(point.Width)
					fac := 1.0 // (float64(point.Pressure) / 204.8)
					pdf.SetLineWidth(w * fac / 5)

					x := float64(lastPoint.X + DeviceWidth/2)
					y := float64(lastPoint.Y)
					dx := float64(point.X + DeviceWidth/2)
					dy := float64(point.Y)
					pdf.Line(x, y, dx, dy)
				}

				lastPoint = point
			}
		}
	}

	logrus.Warn("Drawn all the RECTANNGLESSSSSSSSSSSSSSSS")

	return pdf, nil
}

func RenderCustom(reader io.ReadCloser, output io.Writer) error {
	if output == nil || reader == nil {
		return errors.New("reader or writer were nil")
	}

	pdf, err := parseSceneFile(reader)
	if err != nil {
		return err
	}

	logrus.Warn("WRITING THE THINGGGGGGGGGGGGGGGGGGGG")
	err = pdf.Output(output)
	if err != nil {
		return err
	}

	return nil
}

type SeekCloser struct {
	*bytes.Reader
}

// Close closes
func (*SeekCloser) Close() error {
	return nil
}

func NewSeekCloser(b []byte) io.ReadSeekCloser {

	r := &SeekCloser{
		Reader: bytes.NewReader(b),
	}
	return r
}
