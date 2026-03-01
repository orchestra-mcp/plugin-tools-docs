package toolsdocs

import (
	"context"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/plugin-tools-docs/internal"
	"github.com/orchestra-mcp/plugin-tools-docs/internal/storage"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

// Sender is the interface that the in-process router satisfies.
type Sender interface {
	Send(ctx context.Context, req *pluginv1.PluginRequest) (*pluginv1.PluginResponse, error)
}

// Register adds all 11 documentation tools to the builder.
func Register(builder *plugin.PluginBuilder, sender Sender, workspace string) {
	store := storage.NewDataStorage(sender)
	dp := &internal.DocsPlugin{Storage: store, Workspace: workspace}
	dp.RegisterTools(builder)
}
