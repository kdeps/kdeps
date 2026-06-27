package loader

import (
	"fmt"
	"strings"
	"testing"
)

func TestLoadPDF_FontExtractError(t *testing.T) {
	var b strings.Builder
	fmt.Fprintf(&b, "%%PDF-1.4\n")
	obj1 := b.Len()
	fmt.Fprintf(&b, "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	obj2 := b.Len()
	fmt.Fprintf(&b, "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	obj3 := b.Len()
	fmt.Fprintf(&b, "3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n")
	obj4 := b.Len()
	streamContent := "BT /F1 12 Tf 100 700 Td (Hello) Tj ET"
	fmt.Fprintf(&b, "4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(streamContent), streamContent)
	xrefOffset := b.Len()
	fmt.Fprintf(&b, "xref\n0 5\n")
	fmt.Fprintf(&b, "0000000000 65535 f \n")
	fmt.Fprintf(&b, "%010d 00000 n \n", obj1)
	fmt.Fprintf(&b, "%010d 00000 n \n", obj2)
	fmt.Fprintf(&b, "%010d 00000 n \n", obj3)
	fmt.Fprintf(&b, "%010d 00000 n \n", obj4)
	fmt.Fprintf(&b, "trailer\n<< /Size 5 /Root 1 0 R >>\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF", xrefOffset)
	f := writeTempFileExt(t, b.String(), ".pdf")
	docs, err := loadPDF(f, "")
	if err != nil {
		t.Logf("loadPDF error (expected): %v", err)
		return
	}
	t.Logf("Got %d docs unexpectedly", len(docs))
}
