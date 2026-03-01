// Package tools contains all tool handler implementations for the tools.docs
// plugin. Each function returns a ToolHandler closure that captures the
// DataStorage for data access.
package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToolHandler is an alias for readability.
type ToolHandler = func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error)

// docsDir is the subdirectory within a project that holds documentation files.
const docsDir = "docs"

// docMeta holds the parsed metadata for a documentation page.
type docMeta struct {
	Title     string   `json:"title"`
	Slug      string   `json:"slug"`
	Category  string   `json:"category"`
	Tags      []string `json:"tags"`
	ParentID  string   `json:"parent_id"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// docPath returns the storage path for a doc file: {project_id}/docs/{slug}.md
func docPath(projectID, slug string) string {
	return filepath.Join(projectID, docsDir, slug+".md")
}

// docsPrefix returns the storage prefix for listing all docs in a project:
// {project_id}/docs/
func docsPrefix(projectID string) string {
	return filepath.Join(projectID, docsDir) + string(filepath.Separator)
}

// buildMarkdown constructs a markdown document with a title heading and body.
func buildMarkdown(title, body string) string {
	return fmt.Sprintf("# %s\n\n%s\n", title, body)
}

// parseDocMetadata extracts a docMeta from a structpb.Struct metadata map.
func parseDocMetadata(meta *structpb.Struct) docMeta {
	doc := docMeta{}
	if meta == nil {
		return doc
	}

	fields := meta.GetFields()
	if v, ok := fields["title"]; ok {
		if sv, ok := v.Kind.(*structpb.Value_StringValue); ok {
			doc.Title = sv.StringValue
		}
	}
	if v, ok := fields["slug"]; ok {
		if sv, ok := v.Kind.(*structpb.Value_StringValue); ok {
			doc.Slug = sv.StringValue
		}
	}
	if v, ok := fields["category"]; ok {
		if sv, ok := v.Kind.(*structpb.Value_StringValue); ok {
			doc.Category = sv.StringValue
		}
	}
	if v, ok := fields["parent_id"]; ok {
		if sv, ok := v.Kind.(*structpb.Value_StringValue); ok {
			doc.ParentID = sv.StringValue
		}
	}
	if v, ok := fields["created_at"]; ok {
		if sv, ok := v.Kind.(*structpb.Value_StringValue); ok {
			doc.CreatedAt = sv.StringValue
		}
	}
	if v, ok := fields["updated_at"]; ok {
		if sv, ok := v.Kind.(*structpb.Value_StringValue); ok {
			doc.UpdatedAt = sv.StringValue
		}
	}
	if v, ok := fields["tags"]; ok {
		if lv, ok := v.Kind.(*structpb.Value_ListValue); ok && lv.ListValue != nil {
			for _, item := range lv.ListValue.Values {
				if sv, ok := item.Kind.(*structpb.Value_StringValue); ok {
					doc.Tags = append(doc.Tags, sv.StringValue)
				}
			}
		}
	}

	return doc
}

// docToMetadata converts a docMeta to a structpb.Struct for storage.
func docToMetadata(doc docMeta) (*structpb.Struct, error) {
	m := map[string]any{
		"title":      doc.Title,
		"slug":       doc.Slug,
		"category":   doc.Category,
		"parent_id":  doc.ParentID,
		"created_at": doc.CreatedAt,
		"updated_at": doc.UpdatedAt,
	}
	if len(doc.Tags) > 0 {
		tagList := make([]any, len(doc.Tags))
		for i, t := range doc.Tags {
			tagList[i] = t
		}
		m["tags"] = tagList
	} else {
		m["tags"] = []any{}
	}
	return structpb.NewStruct(m)
}

// formatDocMD formats a single doc page metadata as a Markdown block.
func formatDocMD(doc docMeta) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s\n", doc.Title)
	fmt.Fprintf(&b, "- **Slug:** %s\n", doc.Slug)
	if doc.Category != "" {
		fmt.Fprintf(&b, "- **Category:** %s\n", doc.Category)
	}
	if len(doc.Tags) > 0 {
		fmt.Fprintf(&b, "- **Tags:** %s\n", strings.Join(doc.Tags, ", "))
	}
	if doc.ParentID != "" {
		fmt.Fprintf(&b, "- **Parent:** %s\n", doc.ParentID)
	}
	if doc.CreatedAt != "" {
		fmt.Fprintf(&b, "- **Created:** %s\n", doc.CreatedAt)
	}
	if doc.UpdatedAt != "" {
		fmt.Fprintf(&b, "- **Updated:** %s\n", doc.UpdatedAt)
	}
	return b.String()
}

// formatDocListMD formats a list of docs as a Markdown table.
func formatDocListMD(docs []docMeta, header string) string {
	if len(docs) == 0 {
		return fmt.Sprintf("## %s\n\nNo docs found.\n", header)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", header, len(docs))
	fmt.Fprintf(&b, "| Slug | Title | Category | Tags |\n")
	fmt.Fprintf(&b, "|------|-------|----------|------|\n")
	for _, d := range docs {
		category := d.Category
		if category == "" {
			category = "—"
		}
		tags := "—"
		if len(d.Tags) > 0 {
			tags = strings.Join(d.Tags, ", ")
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", d.Slug, d.Title, category, tags)
	}
	return b.String()
}
