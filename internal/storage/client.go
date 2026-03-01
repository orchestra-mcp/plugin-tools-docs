// Package storage provides an abstraction over the orchestrator's storage
// protocol for reading and writing documentation pages. The StorageClient
// interface allows swapping a real QUIC-based client for an in-memory fake
// during testing.
package storage

import (
	"context"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// StorageClient sends requests to the orchestrator for storage operations.
type StorageClient interface {
	Send(ctx context.Context, req *pluginv1.PluginRequest) (*pluginv1.PluginResponse, error)
}

// DataStorage wraps the storage client for tool handlers.
type DataStorage struct {
	client StorageClient
}

// NewDataStorage creates a new DataStorage with the given client.
func NewDataStorage(client StorageClient) *DataStorage {
	return &DataStorage{client: client}
}

// ReadDoc reads a document from storage at the given path.
func (ds *DataStorage) ReadDoc(ctx context.Context, path string) (*pluginv1.StorageReadResponse, error) {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageRead{
			StorageRead: &pluginv1.StorageReadRequest{
				Path:        path,
				StorageType: "markdown",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	sr := resp.GetStorageRead()
	if sr == nil {
		return nil, fmt.Errorf("unexpected response type for storage read")
	}
	return sr, nil
}

// WriteDoc writes a document to storage at the given path.
func (ds *DataStorage) WriteDoc(ctx context.Context, path string, content []byte, metadata *structpb.Struct, version int64) (*pluginv1.StorageWriteResponse, error) {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageWrite{
			StorageWrite: &pluginv1.StorageWriteRequest{
				Path:            path,
				Content:         content,
				Metadata:        metadata,
				ExpectedVersion: version,
				StorageType:     "markdown",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	sw := resp.GetStorageWrite()
	if sw == nil {
		return nil, fmt.Errorf("unexpected response type for storage write")
	}
	if !sw.Success {
		return nil, fmt.Errorf("storage write failed: %s", sw.Error)
	}
	return sw, nil
}

// DeleteDoc deletes a document from storage at the given path.
func (ds *DataStorage) DeleteDoc(ctx context.Context, path string) (*pluginv1.StorageDeleteResponse, error) {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageDelete{
			StorageDelete: &pluginv1.StorageDeleteRequest{
				Path:        path,
				StorageType: "markdown",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	sd := resp.GetStorageDelete()
	if sd == nil {
		return nil, fmt.Errorf("unexpected response type for storage delete")
	}
	if !sd.Success {
		return nil, fmt.Errorf("storage delete failed")
	}
	return sd, nil
}

// ListDocs lists all documents matching a prefix from storage.
func (ds *DataStorage) ListDocs(ctx context.Context, prefix string) (*pluginv1.StorageListResponse, error) {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageList{
			StorageList: &pluginv1.StorageListRequest{
				Prefix:      prefix,
				Pattern:     "*.md",
				StorageType: "markdown",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	sl := resp.GetStorageList()
	if sl == nil {
		return nil, fmt.Errorf("unexpected response type for storage list")
	}
	return sl, nil
}
