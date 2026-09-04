// Package ingest: PDF text extraction for the capture pipeline.
package ingest

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

// maxPDFTextChars caps extracted text: LLM truncation happens later, but
// embedding a multi-megabyte wall of text is wasteful.
const maxPDFTextChars = 200_000

// ExtractPDFText pulls the text layer out of a PDF, page by page. Pages
// that fail to parse (scanned images, odd encodings) degrade to being
// skipped — a PDF with some readable pages still captures.
func ExtractPDFText(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty pdf")
	}

	reader := bytes.NewReader(data)
	pdfReader, err := pdf.NewReader(reader, int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}

	var out bytes.Buffer
	n := pdfReader.NumPage()
	for i := 1; i <= n; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // skip unreadable page
		}
		if i > 1 {
			out.WriteString("\n\n")
		}
		out.WriteString(text)
		if out.Len() >= maxPDFTextChars {
			break
		}
	}

	if out.Len() == 0 {
		return "", fmt.Errorf("no extractable text (scanned pdf?)")
	}
	result := out.String()
	if len(result) > maxPDFTextChars {
		result = result[:maxPDFTextChars]
	}
	return result, nil
}

// ExtractPDFTextFrom reads a reader fully then extracts.
func ExtractPDFTextFrom(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read pdf: %w", err)
	}
	return ExtractPDFText(data)
}
