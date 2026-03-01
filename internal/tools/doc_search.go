package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-tools-docs/internal/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

// DocSearchSchema returns the JSON Schema for the doc_search tool.
func DocSearchSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug"},
			"query":      map[string]any{"type": "string", "description": "Search query string"},
		},
		"required": []any{"project_id", "query"},
	})
	return s
}

// DocSearch performs a case-insensitive text search across doc titles, categories,
// tags, and body content.
func DocSearch(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "project_id", "query"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		projectID := helpers.GetString(req.Arguments, "project_id")
		query := strings.ToLower(helpers.GetString(req.Arguments, "query"))

		prefix := docsPrefix(projectID)
		listResp, err := store.ListDocs(ctx, prefix)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		var matches []docMeta
		for _, entry := range listResp.Entries {
			base := filepath.Base(entry.Path)
			slug := strings.TrimSuffix(base, ".md")

			readResp, err := store.ReadDoc(ctx, docPath(projectID, slug))
			if err != nil {
				continue
			}
			doc := parseDocMetadata(readResp.Metadata)
			doc.Slug = slug

			// Search in title, category, tags, and body content.
			titleMatch := strings.Contains(strings.ToLower(doc.Title), query)
			categoryMatch := strings.Contains(strings.ToLower(doc.Category), query)
			bodyMatch := strings.Contains(strings.ToLower(string(readResp.Content)), query)
			tagMatch := false
			for _, tag := range doc.Tags {
				if strings.Contains(strings.ToLower(tag), query) {
					tagMatch = true
					break
				}
			}

			if titleMatch || categoryMatch || bodyMatch || tagMatch {
				matches = append(matches, doc)
			}
		}

		header := fmt.Sprintf("Search results for %q", query)
		return helpers.TextResult(formatDocListMD(matches, header)), nil
	}
}
