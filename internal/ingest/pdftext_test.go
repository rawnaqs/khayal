package ingest

import (
	"os"
	"strings"
	"testing"
)

func TestExtractPDFText(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/sample.pdf")
	if err != nil {
		t.Skip("fixture missing")
	}
	text, err := ExtractPDFText(data)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !strings.Contains(strings.ToLower(text), "hello pdf world") {
		t.Errorf("expected fixture text, got: %.200q", text)
	}
}

func TestExtractPDFText_Empty(t *testing.T) {
	if _, err := ExtractPDFText(nil); err == nil {
		t.Error("expected error on empty input")
	}
}

func TestExtractPDFText_Garbage(t *testing.T) {
	if _, err := ExtractPDFText([]byte("this is not a pdf at all")); err == nil {
		t.Error("expected error on garbage input")
	}
}
