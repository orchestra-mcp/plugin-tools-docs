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
			"template": map[string]any{
				"type":        "string",
				"description": "Documentation template to use",
				"enum":        []any{"standard", "api", "guide", "architecture", "runbook"},
			},
			"category": map[string]any{"type": "string", "description": "Category for the doc page (default: derived from template)"},
			"tags": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Tags for the doc page",
			},
		},
		"required": []any{"project_id", "title", "description"},
	})
	return s
}

// DocGenerate creates a documentation page from a description using a standard
// template. Templates provide well-structured sections appropriate for different
// documentation types: standard (general), api (reference), guide (how-to),
// architecture (system design), and runbook (operations).
func DocGenerate(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "project_id", "title", "description"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		projectID := helpers.GetString(req.Arguments, "project_id")
		title := helpers.GetString(req.Arguments, "title")
		description := helpers.GetString(req.Arguments, "description")
		template := helpers.GetString(req.Arguments, "template")
		category := helpers.GetString(req.Arguments, "category")
		tags := helpers.GetStringSlice(req.Arguments, "tags")

		if template == "" {
			template = "standard"
		}

		slug := helpers.Slugify(title)
		now := helpers.NowISO()

		// Generate structured documentation from the template.
		body := generateFromTemplate(template, title, description, now)

		// Derive category from template if not specified.
		if category == "" {
			category = templateCategory(template)
		}

		// Build tags list.
		tagList := []any{"auto-generated", template}
		for _, t := range tags {
			tagList = append(tagList, t)
		}

		meta := map[string]any{
			"title":      title,
			"slug":       slug,
			"category":   category,
			"tags":       tagList,
			"parent_id":  "",
			"created_at": now,
			"updated_at": now,
		}

		metadata, err := structpb.NewStruct(meta)
		if err != nil {
			return helpers.ErrorResult("internal_error", err.Error()), nil
		}

		path := docPath(projectID, slug)
		_, err = store.WriteDoc(ctx, path, []byte(body), metadata, 0)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Generated doc: **%s**\n\n", title)
		fmt.Fprintf(&sb, "- **Slug:** %s\n", slug)
		fmt.Fprintf(&sb, "- **Project:** %s\n", projectID)
		fmt.Fprintf(&sb, "- **Template:** %s\n", template)
		fmt.Fprintf(&sb, "- **Category:** %s\n", category)
		fmt.Fprintf(&sb, "\nThe document has been created with the **%s** template.\n", template)
		fmt.Fprintf(&sb, "Use the /docs skill or an agent to fill in the generated sections with real content.\n")
		return helpers.TextResult(sb.String()), nil
	}
}

// templateCategory returns the default category for a given template type.
func templateCategory(template string) string {
	switch template {
	case "api":
		return "api-reference"
	case "guide":
		return "guides"
	case "architecture":
		return "architecture"
	case "runbook":
		return "operations"
	default:
		return "generated"
	}
}

// generateFromTemplate produces a full markdown document using the specified template.
func generateFromTemplate(template, title, description, now string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)

	switch template {
	case "api":
		fmt.Fprintf(&b, "## Overview\n\n%s\n\n", description)
		fmt.Fprintf(&b, "## Endpoints\n\n")
		fmt.Fprintf(&b, "| Method | Path | Description |\n")
		fmt.Fprintf(&b, "|--------|------|-------------|\n")
		fmt.Fprintf(&b, "| GET | /example | Example endpoint |\n\n")
		fmt.Fprintf(&b, "## Authentication\n\nDescribe the authentication method required for this API.\n\n")
		fmt.Fprintf(&b, "## Request / Response\n\n### Request\n\n```json\n{\n  \"key\": \"value\"\n}\n```\n\n")
		fmt.Fprintf(&b, "### Response\n\n```json\n{\n  \"status\": \"ok\"\n}\n```\n\n")
		fmt.Fprintf(&b, "## Error Codes\n\n")
		fmt.Fprintf(&b, "| Code | Message | Description |\n")
		fmt.Fprintf(&b, "|------|---------|-------------|\n")
		fmt.Fprintf(&b, "| 400 | Bad Request | Invalid input |\n")
		fmt.Fprintf(&b, "| 404 | Not Found | Resource not found |\n\n")
		fmt.Fprintf(&b, "## Rate Limits\n\nDescribe rate limiting policy.\n\n")
		fmt.Fprintf(&b, "---\n_Generated on %s using the **api** template._\n", now)

	case "guide":
		fmt.Fprintf(&b, "## Introduction\n\n%s\n\n", description)
		fmt.Fprintf(&b, "## Prerequisites\n\n- Requirement 1\n- Requirement 2\n\n")
		fmt.Fprintf(&b, "## Step-by-Step Instructions\n\n")
		fmt.Fprintf(&b, "### Step 1: Getting Started\n\nDescribe the first step.\n\n")
		fmt.Fprintf(&b, "### Step 2: Configuration\n\nDescribe the configuration step.\n\n")
		fmt.Fprintf(&b, "### Step 3: Verification\n\nDescribe how to verify it works.\n\n")
		fmt.Fprintf(&b, "## Examples\n\nProvide concrete examples.\n\n")
		fmt.Fprintf(&b, "## Troubleshooting\n\n")
		fmt.Fprintf(&b, "| Problem | Solution |\n")
		fmt.Fprintf(&b, "|---------|----------|\n")
		fmt.Fprintf(&b, "| Issue description | Fix description |\n\n")
		fmt.Fprintf(&b, "## Next Steps\n\nDescribe what the reader can do after completing this guide.\n\n")
		fmt.Fprintf(&b, "---\n_Generated on %s using the **guide** template._\n", now)

	case "architecture":
		fmt.Fprintf(&b, "## Overview\n\n%s\n\n", description)
		fmt.Fprintf(&b, "## Goals & Non-Goals\n\n### Goals\n\n- Goal 1\n\n### Non-Goals\n\n- Non-goal 1\n\n")
		fmt.Fprintf(&b, "## System Design\n\nDescribe the high-level architecture.\n\n")
		fmt.Fprintf(&b, "```\n┌─────────┐     ┌─────────┐     ┌─────────┐\n│ Client  │────▶│ Server  │────▶│   DB    │\n└─────────┘     └─────────┘     └─────────┘\n```\n\n")
		fmt.Fprintf(&b, "## Components\n\n### Component A\n\nDescribe component A.\n\n### Component B\n\nDescribe component B.\n\n")
		fmt.Fprintf(&b, "## Data Flow\n\nDescribe how data flows through the system.\n\n")
		fmt.Fprintf(&b, "## Trade-offs & Decisions\n\n")
		fmt.Fprintf(&b, "| Decision | Rationale | Alternatives Considered |\n")
		fmt.Fprintf(&b, "|----------|-----------|------------------------|\n")
		fmt.Fprintf(&b, "| Decision 1 | Rationale 1 | Alt 1 |\n\n")
		fmt.Fprintf(&b, "## Security Considerations\n\nDescribe security measures.\n\n")
		fmt.Fprintf(&b, "## Future Work\n\nDescribe planned improvements.\n\n")
		fmt.Fprintf(&b, "---\n_Generated on %s using the **architecture** template._\n", now)

	case "runbook":
		fmt.Fprintf(&b, "## Purpose\n\n%s\n\n", description)
		fmt.Fprintf(&b, "## When to Use\n\nDescribe the conditions that trigger this runbook.\n\n")
		fmt.Fprintf(&b, "## Prerequisites\n\n- Access to production environment\n- Required tools installed\n\n")
		fmt.Fprintf(&b, "## Procedure\n\n")
		fmt.Fprintf(&b, "### 1. Assess the Situation\n\n```bash\n# Check current status\necho \"status check\"\n```\n\n")
		fmt.Fprintf(&b, "### 2. Take Action\n\n```bash\n# Execute the fix\necho \"apply fix\"\n```\n\n")
		fmt.Fprintf(&b, "### 3. Verify Resolution\n\n```bash\n# Confirm the fix\necho \"verify\"\n```\n\n")
		fmt.Fprintf(&b, "## Rollback\n\nDescribe how to roll back if the fix fails.\n\n")
		fmt.Fprintf(&b, "## Escalation\n\n| Level | Contact | When |\n")
		fmt.Fprintf(&b, "|-------|---------|------|\n")
		fmt.Fprintf(&b, "| L1 | On-call engineer | First response |\n")
		fmt.Fprintf(&b, "| L2 | Team lead | After 30 min |\n\n")
		fmt.Fprintf(&b, "## Post-Incident\n\n- [ ] Update this runbook\n- [ ] File post-mortem\n- [ ] Update monitoring\n\n")
		fmt.Fprintf(&b, "---\n_Generated on %s using the **runbook** template._\n", now)

	default: // "standard"
		fmt.Fprintf(&b, "## Overview\n\n%s\n\n", description)
		fmt.Fprintf(&b, "## Details\n\nProvide detailed documentation for this topic.\n\n")
		fmt.Fprintf(&b, "## Usage\n\nDescribe how to use this feature or component.\n\n")
		fmt.Fprintf(&b, "### Basic Example\n\n```\n# Example code or command\n```\n\n")
		fmt.Fprintf(&b, "### Advanced Example\n\n```\n# Advanced usage\n```\n\n")
		fmt.Fprintf(&b, "## Configuration\n\nDescribe configuration options.\n\n")
		fmt.Fprintf(&b, "| Option | Type | Default | Description |\n")
		fmt.Fprintf(&b, "|--------|------|---------|-------------|\n")
		fmt.Fprintf(&b, "| option_name | string | \"\" | Description |\n\n")
		fmt.Fprintf(&b, "## Troubleshooting\n\nDescribe common issues and their solutions.\n\n")
		fmt.Fprintf(&b, "---\n_Generated on %s using the **standard** template._\n", now)
	}

	return b.String()
}
