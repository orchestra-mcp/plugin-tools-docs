package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/plugin-tools-docs/internal/storage"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// DocScanSchema returns the JSON Schema for the doc_scan tool.
func DocScanSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug"},
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path to scan for docs (default: docs/). Scanned recursively for .md files.",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Category to assign to all imported docs (default: derived from subdirectory name)",
			},
			"overwrite": map[string]any{
				"type":        "boolean",
				"description": "Overwrite existing docs with the same slug (default: false)",
			},
		},
		"required": []any{"project_id"},
	})
	return s
}

// DocScan scans a workspace directory for markdown files and imports them as
// MCP documentation pages. It walks the directory recursively, reads each .md
// file, extracts a title from the first H1 heading (or filename), and creates
// a doc page in storage.
func DocScan(store *storage.DataStorage, workspace string) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "project_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		projectID := helpers.GetString(req.Arguments, "project_id")
		scanPath := helpers.GetString(req.Arguments, "path")
		categoryOverride := helpers.GetString(req.Arguments, "category")
		overwrite := helpers.GetBool(req.Arguments, "overwrite")

		if scanPath == "" {
			scanPath = "docs"
		}

		// Resolve against workspace root.
		absPath := filepath.Join(workspace, scanPath)

		info, err := os.Stat(absPath)
		if err != nil {
			return helpers.ErrorResult("path_error", fmt.Sprintf("cannot access %s: %v", scanPath, err)), nil
		}
		if !info.IsDir() {
			return helpers.ErrorResult("path_error", fmt.Sprintf("%s is not a directory", scanPath)), nil
		}

		now := helpers.NowISO()
		var imported, skipped, failed int
		var sb strings.Builder
		fmt.Fprintf(&sb, "## Doc Scan Results\n\n")
		fmt.Fprintf(&sb, "**Source:** %s/\n\n", scanPath)
		fmt.Fprintf(&sb, "| File | Title | Slug | Status |\n")
		fmt.Fprintf(&sb, "|------|-------|------|--------|\n")

		err = filepath.Walk(absPath, func(filePath string, fi os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return nil // skip inaccessible entries
			}
			if fi.IsDir() {
				return nil
			}
			if !strings.HasSuffix(fi.Name(), ".md") {
				return nil
			}

			// Read file content.
			content, readErr := os.ReadFile(filePath)
			if readErr != nil {
				failed++
				relPath, _ := filepath.Rel(absPath, filePath)
				fmt.Fprintf(&sb, "| %s | — | — | read error |\n", relPath)
				return nil
			}

			body := string(content)
			relPath, _ := filepath.Rel(absPath, filePath)

			// Extract title from first H1, fallback to filename.
			title := extractTitle(body, fi.Name())
			slug := helpers.Slugify(title)

			// Determine category from subdirectory or override.
			category := categoryOverride
			if category == "" {
				dir := filepath.Dir(relPath)
				if dir != "." {
					// Use the top-level subdirectory as category.
					parts := strings.SplitN(dir, string(filepath.Separator), 2)
					category = parts[0]
				}
			}

			// Check if doc already exists.
			existingPath := docPath(projectID, slug)
			_, readExistErr := store.ReadDoc(ctx, existingPath)
			if readExistErr == nil && !overwrite {
				skipped++
				fmt.Fprintf(&sb, "| %s | %s | %s | skipped (exists) |\n", relPath, title, slug)
				return nil
			}

			// Build metadata.
			meta := map[string]any{
				"title":      title,
				"slug":       slug,
				"category":   category,
				"parent_id":  "",
				"tags":       []any{"scanned"},
				"created_at": now,
				"updated_at": now,
			}

			metadata, structErr := structpb.NewStruct(meta)
			if structErr != nil {
				failed++
				fmt.Fprintf(&sb, "| %s | %s | %s | metadata error |\n", relPath, title, slug)
				return nil
			}

			_, writeErr := store.WriteDoc(ctx, existingPath, content, metadata, 0)
			if writeErr != nil {
				failed++
				fmt.Fprintf(&sb, "| %s | %s | %s | write error |\n", relPath, title, slug)
				return nil
			}

			imported++
			fmt.Fprintf(&sb, "| %s | %s | %s | imported |\n", relPath, title, slug)
			return nil
		})

		if err != nil {
			return helpers.ErrorResult("scan_error", fmt.Sprintf("walk error: %v", err)), nil
		}

		fmt.Fprintf(&sb, "\n**Imported:** %d | **Skipped:** %d | **Failed:** %d\n", imported, skipped, failed)
		return helpers.TextResult(sb.String()), nil
	}
}

// extractTitle pulls the title from the first H1 heading in markdown content.
// Falls back to the filename without extension.
func extractTitle(body, filename string) string {
	for _, line := range strings.SplitN(body, "\n", 20) {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimPrefix(line, "# ")
			title = strings.TrimSpace(title)
			if title != "" {
				return title
			}
		}
	}
	return strings.TrimSuffix(filename, ".md")
}
