package mods

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/blob42/gosuki"
	"github.com/google/go-cmp/cmp"
)

func createTestFile(t *testing.T, content string) (string, error) {
	tmpDir := t.TempDir()

	tmpFile, err := os.CreateTemp(tmpDir, "*.html")
	if err != nil {
		return "", err
	}

	if _, err = tmpFile.WriteString(content); err != nil {
		return "", err
	}

	return filepath.Join(tmpFile.Name()), nil
}

func TestImportBasicParsing(t *testing.T) {
	htmlContent := `
    <!DOCTYPE NETSCAPE-Bookmark-file-1>
    <HTML>
    <HEAD><TITLE>Bookmarks</TITLE></HEAD>
    <BODY>
    <DL><p>
    <DT><A HREF="https://example.com" ADD_DATE="123456789">Example Website</A>
    </DL></p>
    </BODY></HTML>
    `

	tmpFile, err := createTestFile(t, htmlContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpFile)

	bookmarks, err := loadBookmarksFromHTML(tmpFile)
	if err != nil {
		t.Error(err)
	}
	if len(bookmarks) != 1 {
		t.Errorf("Expected 1 bookmark, got %d", len(bookmarks))
		return
	}

	want := &gosuki.Bookmark{
		URL:    "https://example.com",
		Title:  "Example Website",
		Tags:   []string{},
		Module: ImporterID,
	}
	if diff := cmp.Diff(want, bookmarks[0]); diff != "" {
		t.Errorf("Bookmark mismatch (-want +got):\n%s", diff)
	}
}

func TestImportMultipleBookmarks(t *testing.T) {
	htmlContent := `
    <!DOCTYPE NETSCAPE-Bookmark-file-1>
    <HTML>
    <HEAD><TITLE>Bookmarks</TITLE></HEAD>
    <BODY>
    <DL><p>
    <DT><A HREF="https://site1.com">Site 1</A>
    <DT><A HREF="https://site2.com">Site 2</A>
    <DT><A HREF="https://site3.com">Site 3</A>
    </DL></p>
    </BODY></HTML>
    `

	tmpFile, err := createTestFile(t, htmlContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpFile)

	bookmarks, err := loadBookmarksFromHTML(tmpFile)
	if err != nil {
		t.Error(err)
	}
	if len(bookmarks) != 3 {
		t.Errorf("Expected 3 bookmarks, got %d", len(bookmarks))
		return
	}

	expectedTitles := []string{"Site 1", "Site 2", "Site 3"}
	for i, bm := range bookmarks {
		if bm.Title != expectedTitles[i] {
			t.Errorf("Bookmark %d: Expected title %q, got %q", i+1, expectedTitles[i], bm.Title)
		}
	}
}

func TestImportDuplicateURLs(t *testing.T) {
	htmlContent := `
    <!DOCTYPE NETSCAPE-Bookmark-file-1>
    <HTML>
    <HEAD><TITLE>Bookmarks</TITLE></HEAD>
    <BODY>
    <DL><p>
    <DT><A HREF="https://example.com">Example</A>
    <DT><A HREF="https://example.com">Duplicate Example</A>
    </DL></p>
    </BODY></HTML>
    `

	tmpFile, err := createTestFile(t, htmlContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpFile)

	bookmarks, err := loadBookmarksFromHTML(tmpFile)
	if err != nil {
		t.Error(err)
	}
	if len(bookmarks) != 1 {
		t.Errorf("Expected 1 bookmark (duplicate removed), got %d", len(bookmarks))
		return
	}
}

func TestImportInvalidStructure(t *testing.T) {
	htmlContent := `
    <!DOCTYPE NETSCAPE-Bookmark-file-1>
    <HTML>
    <HEAD><TITLE>Bookmarks</TITLE></HEAD>
    <BODY>
    <DL><p>
    <D T><A HREF="https://example.com">Broken Bookmark</A>
    </DL></p>
    </BODY></HTML>
    `

	tmpFile, err := createTestFile(t, htmlContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpFile)

	bookmarks, err := loadBookmarksFromHTML(tmpFile)
	if err != nil {
		t.Error(err)
	}
	if len(bookmarks) != 0 {
		t.Errorf("Expected 0 bookmarks (invalid structure), got %d", len(bookmarks))
		return
	}
}

func TestImportGenerate(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		wantCount   int
	}{
		{
			name: "Basic case",
			htmlContent: `
                <!DOCTYPE NETSCAPE-Bookmark-file-1>
                <HTML>
                <HEAD><TITLE>Bookmarks</TITLE></HEAD>
                <BODY>
                <DL><p>
                <DT><A HREF="https://example.com">Example</A>
                </DL></p>
                </BODY></HTML>
            `,
			wantCount: 1,
		},
		{
			name: "Bookmark in subfolder",
			htmlContent: `
                <!DOCTYPE NETSCAPE-Bookmark-file-1>
                <HTML>
                <HEAD><TITLE>Bookmarks</TITLE></HEAD>
                <BODY>
                <DL><p>
                <DT><H3 ADDITIONAL_INFO="Folder">My Folder</H3>
                <DL><p>
                    <DT><A HREF="https://example.com">Bookmark in folder</A>
                </DL></p>
                </DL></p>
                </BODY></HTML>
            `,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := createTestFile(t, tt.htmlContent)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			defer os.Remove(tmpFile)

			got, err := loadBookmarksFromHTML(tmpFile)
			if err != nil {
				t.Error(err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("loadBookmarksFromHTML() len = %d, want %d", len(got), tt.wantCount)
			}

			// Additional checks for tags and folders
			for _, bm := range got {
				switch tt.name {
				case "Bookmark with tags":
					if !slices.Contains(bm.Tags, "tag1") || !slices.Contains(bm.Tags, "tag2") {
						t.Errorf("Expected tags 'tag1' and 'tag2', got %q", bm.Tags)
					}
				case "Bookmark in subfolder":
					if !slices.Contains(bm.Tags, "My Folder") {
						t.Errorf("Expected tag 'My Folder', got %q", bm.Tags)
					}
				}
			}
		})
	}
}
