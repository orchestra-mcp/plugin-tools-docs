package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"github.com/orchestra-mcp/plugin-tools-docs/internal/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

// DocExportSchema returns the JSON Schema for the doc_export tool.
func DocExportSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug"},
		},
		"required": []any{"project_id"},
	})
	return s
}

// docForExport holds metadata and body content together for sorting and export.
type docForExport struct {
	meta docMeta
	body string
}

// DocExport exports all docs in a project as a single concatenated markdown
// document, separated by horizontal rules.
func DocExport(store *storage.DataStorage) ToolHandler {
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

		if len(listResp.Entries) == 0 {
			return helpers.TextResult(fmt.Sprintf("No docs found in project %s", projectID)), nil
		}

		var allDocs []docForExport
		for _, entry := range listResp.Entries {
			base := filepath.Base(entry.Path)
			slug := strings.TrimSuffix(base, ".md")

			readResp, err := store.ReadDoc(ctx, docPath(projectID, slug))
			if err != nil {
				continue
			}
			doc := parseDocMetadata(readResp.Metadata)
			doc.Slug = slug

			allDocs = append(allDocs, docForExport{
				meta: doc,
				body: string(readResp.Content),
			})
		}

		// Sort by title for consistent output.
		sort.Slice(allDocs, func(i, j int) bool {
			return allDocs[i].meta.Title < allDocs[j].meta.Title
		})

		var sb strings.Builder
		fmt.Fprintf(&sb, "# %s — Documentation Export\n\n", projectID)
		fmt.Fprintf(&sb, "_Exported %d pages_\n\n", len(allDocs))
		fmt.Fprintf(&sb, "---\n\n")

		for i, d := range allDocs {
			// Write frontmatter-style header for each doc.
			fmt.Fprintf(&sb, "## %s\n\n", d.meta.Title)
			if d.meta.Category != "" {
				fmt.Fprintf(&sb, "**Category:** %s", d.meta.Category)
				if len(d.meta.Tags) > 0 {
					fmt.Fprintf(&sb, " | **Tags:** %s", strings.Join(d.meta.Tags, ", "))
				}
				fmt.Fprintf(&sb, "\n\n")
			} else if len(d.meta.Tags) > 0 {
				fmt.Fprintf(&sb, "**Tags:** %s\n\n", strings.Join(d.meta.Tags, ", "))
			}

			sb.WriteString(d.body)

			if i < len(allDocs)-1 {
				fmt.Fprintf(&sb, "\n\n---\n\n")
			}
		}

		return helpers.TextResult(sb.String()), nil
	}
}
