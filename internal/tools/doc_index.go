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

// DocIndexSchema returns the JSON Schema for the doc_index tool.
func DocIndexSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug"},
		},
		"required": []any{"project_id"},
	})
	return s
}

// DocIndex reads all docs for a project and returns a summary index with slug,
// title, and category for each page. This is useful for building search indexes
// or generating a table of contents.
func DocIndex(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "project_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		projectID := helpers.GetString(req.Arguments, "project_id")

		prefix := docsPrefix(projectID)
		listResp, err := store.ListDocs(ctx, prefix)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		var indexed int
		var sb strings.Builder
		fmt.Fprintf(&sb, "## Doc Index for %s\n\n", projectID)
		fmt.Fprintf(&sb, "| Slug | Title | Category | Tags |\n")
		fmt.Fprintf(&sb, "|------|-------|----------|------|\n")

		for _, entry := range listResp.Entries {
			base := filepath.Base(entry.Path)
			slug := strings.TrimSuffix(base, ".md")

			readResp, err := store.ReadDoc(ctx, docPath(projectID, slug))
			if err != nil {
				continue
			}
			doc := parseDocMetadata(readResp.Metadata)
			doc.Slug = slug

			tags := "—"
			if len(doc.Tags) > 0 {
				tags = strings.Join(doc.Tags, ", ")
			}
			category := doc.Category
			if category == "" {
				category = "—"
			}

			fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n", slug, doc.Title, category, tags)
			indexed++
		}

		fmt.Fprintf(&sb, "\n**Total indexed:** %d pages\n", indexed)
		return helpers.TextResult(sb.String()), nil
	}
}
