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

// DocTreeSchema returns the JSON Schema for the doc_tree tool.
func DocTreeSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug"},
		},
		"required": []any{"project_id"},
	})
	return s
}

// treeNode represents a page in the doc tree hierarchy.
type treeNode struct {
	doc      docMeta
	children []*treeNode
}

// DocTree returns the full nested tree structure of all docs in a project,
// organized by parent-child relationships.
func DocTree(store *storage.DataStorage) ToolHandler {
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

		// Collect all docs.
		allDocs := make(map[string]docMeta)
		for _, entry := range listResp.Entries {
			base := filepath.Base(entry.Path)
			slug := strings.TrimSuffix(base, ".md")

			readResp, err := store.ReadDoc(ctx, docPath(projectID, slug))
			if err != nil {
				continue
			}
			doc := parseDocMetadata(readResp.Metadata)
			doc.Slug = slug
			allDocs[slug] = doc
		}

		// Build tree: group by parent_id.
		nodeMap := make(map[string]*treeNode)
		for slug, doc := range allDocs {
			nodeMap[slug] = &treeNode{doc: doc}
		}

		var roots []*treeNode
		for slug, node := range nodeMap {
			parentID := allDocs[slug].ParentID
			if parentID == "" || nodeMap[parentID] == nil {
				roots = append(roots, node)
			} else {
				nodeMap[parentID].children = append(nodeMap[parentID].children, node)
			}
		}

		// Sort roots and children alphabetically by title.
		sort.Slice(roots, func(i, j int) bool {
			return roots[i].doc.Title < roots[j].doc.Title
		})
		for _, node := range nodeMap {
			sort.Slice(node.children, func(i, j int) bool {
				return node.children[i].doc.Title < node.children[j].doc.Title
			})
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "## Doc Tree for %s\n\n", projectID)

		if len(roots) == 0 {
			fmt.Fprintf(&sb, "No docs found.\n")
		} else {
			for _, root := range roots {
				renderTree(&sb, root, 0)
			}
		}

		return helpers.TextResult(sb.String()), nil
	}
}

// renderTree recursively renders a tree node as indented markdown.
func renderTree(sb *strings.Builder, node *treeNode, depth int) {
	indent := strings.Repeat("  ", depth)
	category := ""
	if node.doc.Category != "" {
		category = fmt.Sprintf(" [%s]", node.doc.Category)
	}
	fmt.Fprintf(sb, "%s- **%s** (`%s`)%s\n", indent, node.doc.Title, node.doc.Slug, category)
	for _, child := range node.children {
		renderTree(sb, child, depth+1)
	}
}
