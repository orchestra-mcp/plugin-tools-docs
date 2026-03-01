// Package internal contains the core registration logic for the tools.docs
// plugin. The DocsPlugin struct wires all 10 tool handlers to the plugin
// builder with their schemas and descriptions.
package internal

import (
	"github.com/orchestra-mcp/sdk-go/plugin"
	"github.com/orchestra-mcp/plugin-tools-docs/internal/storage"
	"github.com/orchestra-mcp/plugin-tools-docs/internal/tools"
)

// DocsPlugin holds the shared dependencies for all tool handlers.
type DocsPlugin struct {
	Storage *storage.DataStorage
}

// RegisterTools registers all 10 documentation tools on the given plugin builder.
func (dp *DocsPlugin) RegisterTools(builder *plugin.PluginBuilder) {
	s := dp.Storage

	// --- CRUD tools (4) ---
	builder.RegisterTool("doc_create",
		"Create a new wiki/documentation page in a project",
		tools.DocCreateSchema(), tools.DocCreate(s))
	builder.RegisterTool("doc_get",
		"Get a documentation page by slug",
		tools.DocGetSchema(), tools.DocGet(s))
	builder.RegisterTool("doc_update",
		"Update the content and metadata of a documentation page",
		tools.DocUpdateSchema(), tools.DocUpdate(s))
	builder.RegisterTool("doc_delete",
		"Delete a documentation page",
		tools.DocDeleteSchema(), tools.DocDelete(s))

	// --- Query tools (2) ---
	builder.RegisterTool("doc_list",
		"List documentation pages in a project, optionally filtered by category or parent",
		tools.DocListSchema(), tools.DocList(s))
	builder.RegisterTool("doc_search",
		"Search documentation pages by query across titles, categories, tags, and content",
		tools.DocSearchSchema(), tools.DocSearch(s))

	// --- Generation and indexing tools (2) ---
	builder.RegisterTool("doc_generate",
		"Generate a documentation page from a code description",
		tools.DocGenerateSchema(), tools.DocGenerate(s))
	builder.RegisterTool("doc_index",
		"Index all documentation pages for a project and return a summary table",
		tools.DocIndexSchema(), tools.DocIndex(s))

	// --- Structure and export tools (2) ---
	builder.RegisterTool("doc_tree",
		"Get the full nested tree structure of all documentation pages",
		tools.DocTreeSchema(), tools.DocTree(s))
	builder.RegisterTool("doc_export",
		"Export all documentation pages as a single concatenated markdown document",
		tools.DocExportSchema(), tools.DocExport(s))
}
