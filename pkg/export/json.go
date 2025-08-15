package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/blob42/gosuki"
)

type JSONExporter struct{}

func (je *JSONExporter) WriteHeader(w io.Writer) error {
	_, err := w.Write([]byte("["))
	return err
}

func (je *JSONExporter) WriteFooter(w io.Writer) error {
	_, err := w.Write([]byte("]"))
	return err
}

func (je *JSONExporter) ExportBookmarks(bookmarks []*gosuki.Bookmark, w io.Writer) error {
	pinboardBookmarks := make([]pinboardBookmark, 0, len(bookmarks))
	for _, book := range bookmarks {
		pinboardBookmarks = append(pinboardBookmarks, jsonBookmark(book))
	}

	return json.NewEncoder(w).Encode(pinboardBookmarks)
}

type pinboardBookmark struct {
	Href        string `json:"href"`
	Description string `json:"description"`
	Extended    string `json:"extended"`
	Meta        string `json:"meta"`
	Hash        string `json:"hash"`
	Time        string `json:"time"`
	Shared      string `json:"shared"`
	Toread      string `json:"toread"`
	Tags        string `json:"tags"`
}

func jsonBookmark(book *gosuki.Bookmark) pinboardBookmark {
	timeStr := time.Unix(int64(book.Modified), 0).UTC().Format(time.RFC3339)

	return pinboardBookmark{
		Href:        book.URL,
		Description: book.Title,
		Extended:    "",
		Meta:        "",
		Hash:        book.Xhsum,
		Time:        timeStr,
		Shared:      "no",
		Toread:      "no",
		Tags:        strings.Join(book.Tags, ","),
	}
}

func (je *JSONExporter) MarshalBookmark(book *gosuki.Bookmark) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(jsonBookmark(book)); err != nil {
		panic(fmt.Sprintf("encoding %v", book))
	}

	buf.Truncate(buf.Len() - 1)
	return buf.Bytes()
}

var _ Exporter = (*JSONExporter)(nil)
