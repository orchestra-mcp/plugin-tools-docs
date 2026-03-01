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

// DocGenerateSchema returns the JSON Schema for the doc_generate tool.
func DocGenerateSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id":  map[string]any{"type": "string", "description": "Project slug"},
			"title":       map[string]any{"type": "string", "description": "Document title"},
			"description": map[string]any{"type": "string", "description": "Description of what to document (used to generate content)"},
		},
		"required": []any{"project_id", "title", "description"},
	})
	return s
}

// DocGenerate creates a documentation page from a code description. It generates
// a structured markdown document with sections for overview, details, usage, and
// notes based on the provided description.
func DocGenerate(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "project_id", "title", "description"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		projectID := helpers.GetString(req.Arguments, "project_id")
		title := helpers.GetString(req.Arguments, "title")
		description := helpers.GetString(req.Arguments, "description")

		slug := helpers.Slugify(title)
		now := helpers.NowISO()

		// Generate structured documentation from the description.
		var body strings.Builder
		fmt.Fprintf(&body, "# %s\n\n", title)
		fmt.Fprintf(&body, "## Overview\n\n%s\n\n", description)
		fmt.Fprintf(&body, "## Details\n\n_TODO: Add detailed documentation._\n\n")
		fmt.Fprintf(&body, "## Usage\n\n_TODO: Add usage examples._\n\n")
		fmt.Fprintf(&body, "## Notes\n\n- Generated from description on %s\n", now)

		meta := map[string]any{
			"title":      title,
			"slug":       slug,
			"category":   "generated",
			"tags":       []any{"auto-generated"},
			"parent_id":  "",
			"created_at": now,
			"updated_at": now,
		}

		metadata, err := structpb.NewStruct(meta)
		if err != nil {
			return helpers.ErrorResult("internal_error", err.Error()), nil
		}

		path := docPath(projectID, slug)
		_, err = store.WriteDoc(ctx, path, []byte(body.String()), metadata, 0)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Generated doc: **%s**\n\n", title)
		fmt.Fprintf(&sb, "- **Slug:** %s\n", slug)
		fmt.Fprintf(&sb, "- **Project:** %s\n", projectID)
		fmt.Fprintf(&sb, "- **Category:** generated\n")
		fmt.Fprintf(&sb, "\nThe document has been created with Overview, Details, Usage, and Notes sections.\n")
		return helpers.TextResult(sb.String()), nil
	}
}
