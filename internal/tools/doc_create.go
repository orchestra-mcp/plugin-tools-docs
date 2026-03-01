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

// DocCreateSchema returns the JSON Schema for the doc_create tool.
func DocCreateSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug"},
			"title":      map[string]any{"type": "string", "description": "Page title"},
			"body":       map[string]any{"type": "string", "description": "Page content (markdown)"},
			"category":   map[string]any{"type": "string", "description": "Category (e.g. architecture, guides)"},
			"tags": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Tags for the page",
			},
			"parent_id": map[string]any{"type": "string", "description": "Parent page slug for nesting"},
		},
		"required": []any{"project_id", "title", "body"},
	})
	return s
}

// DocCreate creates a new wiki page in a project.
func DocCreate(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "project_id", "title", "body"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		projectID := helpers.GetString(req.Arguments, "project_id")
		title := helpers.GetString(req.Arguments, "title")
		body := helpers.GetString(req.Arguments, "body")
		category := helpers.GetString(req.Arguments, "category")
		parentID := helpers.GetString(req.Arguments, "parent_id")
		tags := helpers.GetStringSlice(req.Arguments, "tags")

		slug := helpers.Slugify(title)
		now := helpers.NowISO()

		// Build frontmatter metadata.
		meta := map[string]any{
			"title":      title,
			"slug":       slug,
			"category":   category,
			"parent_id":  parentID,
			"created_at": now,
			"updated_at": now,
		}
		if len(tags) > 0 {
			tagList := make([]any, len(tags))
			for i, t := range tags {
				tagList[i] = t
			}
			meta["tags"] = tagList
		} else {
			meta["tags"] = []any{}
		}

		metadata, err := structpb.NewStruct(meta)
		if err != nil {
			return helpers.ErrorResult("internal_error", err.Error()), nil
		}

		// Build the full markdown content with frontmatter.
		content := buildMarkdown(title, body)
		path := docPath(projectID, slug)

		_, err = store.WriteDoc(ctx, path, []byte(content), metadata, 0)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Created doc page: **%s**\n\n", title)
		fmt.Fprintf(&sb, "- **Slug:** %s\n", slug)
		fmt.Fprintf(&sb, "- **Project:** %s\n", projectID)
		if category != "" {
			fmt.Fprintf(&sb, "- **Category:** %s\n", category)
		}
		if len(tags) > 0 {
			fmt.Fprintf(&sb, "- **Tags:** %s\n", strings.Join(tags, ", "))
		}
		if parentID != "" {
			fmt.Fprintf(&sb, "- **Parent:** %s\n", parentID)
		}
		return helpers.TextResult(sb.String()), nil
	}
}
