package parsing

import (
	"reflect"
	"strings"
	"testing"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/tree"
	"github.com/stretchr/testify/assert"
)

func TestParseTags(t *testing.T) {
	logging.SetLevel(logging.Silent)
	tests := []struct {
		name         string
		title        string
		initialTags  []string
		expectedTags []string
		expectError  bool
	}{
		{
			name:         "Valid tags in node title",
			title:        "This is a #test and #example",
			initialTags:  nil,
			expectedTags: []string{"test", "example"},
			expectError:  false,
		},
		{
			name:         "Valid tags with hyphen and underscore",
			title:        "#foo-tag and #bar_tag",
			initialTags:  nil,
			expectedTags: []string{"foo-tag", "bar_tag"},
			expectError:  false,
		},
		{
			name:         "Invalid characters in tag",
			title:        "#tag!@#bar",
			initialTags:  nil,
			expectedTags: []string{"tag", "bar"},
			expectError:  false,
		},
		{
			name:         "No tags",
			title:        "No tags here",
			initialTags:  nil,
			expectedTags: []string{},
			expectError:  false,
		},
		{
			name:         "Tag at the end of the string",
			title:        "this is a end of sentence #tag",
			initialTags:  nil,
			expectedTags: []string{"tag"},
			expectError:  false,
		},
		{
			name:         "Multiple tags in one string",
			title:        "#test1 #test2 #test3",
			initialTags:  nil,
			expectedTags: []string{"test1", "test2", "test3"},
			expectError:  false,
		},
		{
			name:         "Tag with dot",
			title:        "#tag.with.dot",
			initialTags:  nil,
			expectedTags: []string{"tag.with.dot"},
			expectError:  false,
		},
		{
			name:         "Appending to existing tags",
			title:        "#newtag",
			initialTags:  []string{"existing"},
			expectedTags: []string{"existing", "newtag"},
			expectError:  false,
		},
		{
			name:         "Appending action to existing tags",
			title:        "@action",
			initialTags:  []string{"existing"},
			expectedTags: []string{"existing", "@action"},
			expectError:  false,
		},
		{
			name:         "Valid action tags in node title",
			title:        "This is a @action and #example",
			initialTags:  nil,
			expectedTags: []string{"example", "@action"},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Create two subtests for each type
			t.Run("tree.Node", func(t *testing.T) {
				item := &tree.Node{Title: tt.title, Type: tree.URLNode}
				item.Tags = nil
				if tt.initialTags != nil {
					item.Tags = tt.initialTags
				}
				err := parseTags(item)
				if tt.expectError {
					if err == nil {
						t.Errorf("expected error, got nil")
					}
					return
				}
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(item.Tags, tt.expectedTags) {
					t.Errorf("expected tags %v, got %v", tt.expectedTags, item.Tags)
				}
				// Check that the title no longer contains the matched tags
				for _, tag := range tt.expectedTags {
					if strings.Contains(item.Title, "#"+tag) {
						t.Errorf("title still contains tag %s after parsing", tag)
					}
				}
			})

			t.Run("gosuki.Bookmark", func(t *testing.T) {
				item := &gosuki.Bookmark{Title: tt.title}
				item.Tags = nil
				if tt.initialTags != nil {
					item.Tags = tt.initialTags
				}
				err := parseTags(item)
				if tt.expectError {
					if err == nil {
						t.Errorf("expected error, got nil")
					}
					return
				}
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(item.Tags, tt.expectedTags) {
					t.Errorf("expected tags %v, got %v", tt.expectedTags, item.Tags)
				}
				// Check that the title no longer contains the matched tags
				for _, tag := range tt.expectedTags {
					if strings.Contains(item.Title, "#"+tag) {
						t.Errorf("title still contains tag %s after parsing", tag)
					}
				}
			})
		})
	}

	t.Run("unsupported type", func(t *testing.T) {
		item := "unsupported tag"
		err := parseTags(item)
		assert.Error(t, err)
	})
}
