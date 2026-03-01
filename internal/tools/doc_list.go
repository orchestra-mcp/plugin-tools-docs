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

// DocListSchema returns the JSON Schema for the doc_list tool.
func DocListSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug"},
			"category":   map[string]any{"type": "string", "description": "Filter by category"},
			"parent_id":  map[string]any{"type": "string", "description": "Filter by parent slug"},
		},
		"required": []any{"project_id"},
	})
	return s
}

// DocList lists all wiki pages in a project, optionally filtered by category or parent.
func DocList(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "project_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		projectID := helpers.GetString(req.Arguments, "project_id")
		categoryFilter := helpers.GetString(req.Arguments, "category")
		parentFilter := helpers.GetString(req.Arguments, "parent_id")

		prefix := docsPrefix(projectID)
		listResp, err := store.ListDocs(ctx, prefix)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		var docs []docMeta
		for _, entry := range listResp.Entries {
			base := filepath.Base(entry.Path)
			slug := strings.TrimSuffix(base, ".md")

			readResp, err := store.ReadDoc(ctx, docPath(projectID, slug))
			if err != nil {
				continue
			}
			doc := parseDocMetadata(readResp.Metadata)
			doc.Slug = slug

			// Apply filters.
			if categoryFilter != "" && doc.Category != categoryFilter {
				continue
			}
			if parentFilter != "" && doc.ParentID != parentFilter {
				continue
			}

			docs = append(docs, doc)
		}

		header := "Docs"
		if categoryFilter != "" {
			header = fmt.Sprintf("Docs (%s)", categoryFilter)
		}

		return helpers.TextResult(formatDocListMD(docs, header)), nil
	}
}
