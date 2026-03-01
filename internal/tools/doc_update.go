package tools

import (
	"context"
	"fmt"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-tools-docs/internal/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

// DocUpdateSchema returns the JSON Schema for the doc_update tool.
func DocUpdateSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug"},
			"slug":       map[string]any{"type": "string", "description": "Page slug"},
			"body":       map[string]any{"type": "string", "description": "New page content (markdown)"},
			"title":      map[string]any{"type": "string", "description": "New title"},
			"category":   map[string]any{"type": "string", "description": "New category"},
			"tags": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "New tags",
			},
		},
		"required": []any{"project_id", "slug", "body"},
	})
	return s
}

// DocUpdate updates the content and metadata of an existing wiki page.
func DocUpdate(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "project_id", "slug", "body"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		projectID := helpers.GetString(req.Arguments, "project_id")
		slug := helpers.GetString(req.Arguments, "slug")
		body := helpers.GetString(req.Arguments, "body")

		path := docPath(projectID, slug)
		existing, err := store.ReadDoc(ctx, path)
		if err != nil {
			return helpers.ErrorResult("not_found", fmt.Sprintf("doc %q not found in project %q", slug, projectID)), nil
		}

		doc := parseDocMetadata(existing.Metadata)

		// Apply optional updates.
		if t := helpers.GetString(req.Arguments, "title"); t != "" {
			doc.Title = t
		}
		if c := helpers.GetString(req.Arguments, "category"); c != "" {
			doc.Category = c
		}
		if tags := helpers.GetStringSlice(req.Arguments, "tags"); tags != nil {
			doc.Tags = tags
		}
		doc.UpdatedAt = helpers.NowISO()

		metadata, err := docToMetadata(doc)
		if err != nil {
			return helpers.ErrorResult("internal_error", err.Error()), nil
		}

		content := buildMarkdown(doc.Title, body)
		_, err = store.WriteDoc(ctx, path, []byte(content), metadata, existing.Version)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Updated doc: **%s**\n\n", doc.Title)
		fmt.Fprintf(&sb, "- **Slug:** %s\n", slug)
		fmt.Fprintf(&sb, "- **Project:** %s\n", projectID)
		return helpers.TextResult(sb.String()), nil
	}
}
