package tools

import (
	"context"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-tools-docs/internal/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

// DocGetSchema returns the JSON Schema for the doc_get tool.
func DocGetSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug"},
			"slug":       map[string]any{"type": "string", "description": "Page slug"},
		},
		"required": []any{"project_id", "slug"},
	})
	return s
}

// DocGet retrieves a wiki page by project and slug.
func DocGet(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "project_id", "slug"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		projectID := helpers.GetString(req.Arguments, "project_id")
		slug := helpers.GetString(req.Arguments, "slug")

		path := docPath(projectID, slug)
		resp, err := store.ReadDoc(ctx, path)
		if err != nil {
			return helpers.ErrorResult("not_found", fmt.Sprintf("doc %q not found in project %q", slug, projectID)), nil
		}

		doc := parseDocMetadata(resp.Metadata)
		md := formatDocMD(doc) + "\n---\n\n" + string(resp.Content)
		return helpers.TextResult(md), nil
	}
}
