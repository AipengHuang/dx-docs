package session

import (
	"context"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/infrastructure/docparser"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type fullContentFileService struct {
	interfaces.FileService
}

func (fullContentFileService) SaveBytes(context.Context, []byte, uint64, string, bool) (string, error) {
	return "local://attachments/attachment_test.txt", nil
}

// A document longer than the inline line limit must keep its complete text in
// FullContent so oversized-attachment ingestion can index the whole file.
func TestTruncatedAttachmentKeepsFullContentForIngestion(t *testing.T) {
	lines := make([]string, 0, 600)
	for i := 0; i < 600; i++ {
		lines = append(lines, "row-"+strings.Repeat("x", 3)+"-"+itoa(i))
	}
	content := strings.Join(lines, "\n")

	processor := NewAttachmentProcessor(fullContentFileService{}, nil, docparser.NewImageResolver(), nil)
	attachment, err := processor.ProcessAttachment(context.Background(), []byte(content), "big.txt", int64(len(content)), 10000, "")
	if err != nil {
		t.Fatalf("process failed: %v", err)
	}
	if !attachment.IsTruncated || attachment.LineCount != 600 {
		t.Fatalf("expected truncation of 600 lines, got truncated=%v lines=%d", attachment.IsTruncated, attachment.LineCount)
	}
	if got := strings.Count(attachment.Content, "\n"); got != maxTextFileLines-1 {
		t.Fatalf("inline content kept %d lines, want %d", got+1, maxTextFileLines)
	}
	if attachment.FullContent != content {
		t.Fatalf("FullContent does not preserve the whole document: len=%d want %d", len(attachment.FullContent), len(content))
	}
	if attachment.TotalChunks != 0 || attachment.ContentMode != "" {
		t.Fatalf("unexpected chunk metadata on inline attachment: %+v", attachment)
	}
}

// Short documents stay fully inline and must not carry a redundant copy.
func TestSmallAttachmentHasNoFullContentCopy(t *testing.T) {
	processor := NewAttachmentProcessor(fullContentFileService{}, nil, docparser.NewImageResolver(), nil)
	attachment, err := processor.ProcessAttachment(context.Background(), []byte("short file"), "small.txt", 10, 10000, "")
	if err != nil {
		t.Fatalf("process failed: %v", err)
	}
	if attachment.IsTruncated || attachment.FullContent != "" || attachment.Content != "short file" {
		t.Fatalf("small attachment should be fully inline: %+v", attachment)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
