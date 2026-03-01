package tools_test

import (
	"context"
	"strings"
	"testing"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/plugin-tools-docs/internal/storage"
	"github.com/orchestra-mcp/plugin-tools-docs/internal/tools"
	"google.golang.org/protobuf/types/known/structpb"
)

// ---------- Mock storage client ----------

type docRecord struct {
	metadata *structpb.Struct
	content  []byte
	version  int64
}

type mockClient struct {
	docs    map[string]*docRecord
	version int64
}

func newMockClient() *mockClient {
	return &mockClient{docs: make(map[string]*docRecord)}
}

func (m *mockClient) Send(_ context.Context, req *pluginv1.PluginRequest) (*pluginv1.PluginResponse, error) {
	switch r := req.Request.(type) {
	case *pluginv1.PluginRequest_StorageWrite:
		m.version++
		m.docs[r.StorageWrite.Path] = &docRecord{
			metadata: r.StorageWrite.Metadata,
			content:  r.StorageWrite.Content,
			version:  m.version,
		}
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageWrite{
				StorageWrite: &pluginv1.StorageWriteResponse{
					Success:    true,
					NewVersion: m.version,
				},
			},
		}, nil

	case *pluginv1.PluginRequest_StorageRead:
		rec, ok := m.docs[r.StorageRead.Path]
		if !ok {
			return &pluginv1.PluginResponse{
				Response: &pluginv1.PluginResponse_StorageRead{
					StorageRead: &pluginv1.StorageReadResponse{},
				},
			}, nil
		}
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageRead{
				StorageRead: &pluginv1.StorageReadResponse{
					Metadata: rec.metadata,
					Content:  rec.content,
					Version:  rec.version,
				},
			},
		}, nil

	case *pluginv1.PluginRequest_StorageDelete:
		delete(m.docs, r.StorageDelete.Path)
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageDelete{
				StorageDelete: &pluginv1.StorageDeleteResponse{Success: true},
			},
		}, nil

	case *pluginv1.PluginRequest_StorageList:
		prefix := r.StorageList.Prefix
		var entries []*pluginv1.StorageEntry
		for path := range m.docs {
			if strings.HasPrefix(path, prefix) {
				entries = append(entries, &pluginv1.StorageEntry{Path: path})
			}
		}
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageList{
				StorageList: &pluginv1.StorageListResponse{Entries: entries},
			},
		}, nil
	}
	return &pluginv1.PluginResponse{}, nil
}

// ---------- Helpers ----------

func makeStore() *storage.DataStorage {
	return storage.NewDataStorage(newMockClient())
}

func makeArgs(t *testing.T, m map[string]any) *structpb.Struct {
	t.Helper()
	s, err := structpb.NewStruct(m)
	if err != nil {
		t.Fatalf("makeArgs: %v", err)
	}
	return s
}

func assertSuccess(t *testing.T, resp *pluginv1.ToolResponse) {
	t.Helper()
	if !resp.Success {
		t.Fatalf("expected success, got error: %s — %s", resp.ErrorCode, resp.ErrorMessage)
	}
}

func assertError(t *testing.T, resp *pluginv1.ToolResponse, code string) {
	t.Helper()
	if resp.Success {
		t.Fatalf("expected error code %q but got success", code)
	}
	if resp.ErrorCode != code {
		t.Fatalf("expected error code %q, got %q", code, resp.ErrorCode)
	}
}

func responseText(t *testing.T, resp *pluginv1.ToolResponse) string {
	t.Helper()
	if resp.Result == nil {
		return ""
	}
	v, ok := resp.Result.Fields["text"]
	if !ok {
		t.Fatalf("result missing 'text' key")
	}
	return v.GetStringValue()
}

// ---------- doc_create ----------

func TestDocCreate_Basic(t *testing.T) {
	store := makeStore()
	fn := tools.DocCreate(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "my-project",
			"title":      "Getting Started",
			"body":       "Welcome to the project.",
		}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSuccess(t, resp)
	text := responseText(t, resp)
	if !strings.Contains(text, "Getting Started") {
		t.Fatalf("expected title in response, got: %s", text)
	}
	if !strings.Contains(text, "getting-started") {
		t.Fatalf("expected slug in response, got: %s", text)
	}
}

func TestDocCreate_WithCategory(t *testing.T) {
	store := makeStore()
	fn := tools.DocCreate(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"title":      "Architecture Overview",
			"body":       "Describe the arch.",
			"category":   "architecture",
		}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSuccess(t, resp)
	text := responseText(t, resp)
	if !strings.Contains(text, "architecture") {
		t.Fatalf("expected category in response, got: %s", text)
	}
}

func TestDocCreate_MissingProjectID(t *testing.T) {
	store := makeStore()
	fn := tools.DocCreate(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"title": "No Project",
			"body":  "Body",
		}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertError(t, resp, "validation_error")
}

func TestDocCreate_MissingTitle(t *testing.T) {
	store := makeStore()
	fn := tools.DocCreate(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"body":       "Body",
		}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertError(t, resp, "validation_error")
}

// ---------- doc_get ----------

func TestDocGet_Exists(t *testing.T) {
	store := makeStore()
	// Create first
	tools.DocCreate(store)(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"title":      "API Reference",
			"body":       "API docs here.",
		}),
	})

	fn := tools.DocGet(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"slug":       "api-reference",
		}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSuccess(t, resp)
	text := responseText(t, resp)
	if !strings.Contains(text, "API Reference") {
		t.Fatalf("expected title in get response, got: %s", text)
	}
}

func TestDocGet_MissingSlug(t *testing.T) {
	store := makeStore()
	fn := tools.DocGet(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{"project_id": "proj"}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertError(t, resp, "validation_error")
}

// ---------- doc_list ----------

func TestDocList_Empty(t *testing.T) {
	store := makeStore()
	fn := tools.DocList(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{"project_id": "empty-proj"}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSuccess(t, resp)
	text := responseText(t, resp)
	if !strings.Contains(text, "No docs found") {
		t.Fatalf("expected no docs message, got: %s", text)
	}
}

func TestDocList_MultiplePages(t *testing.T) {
	store := makeStore()
	createFn := tools.DocCreate(store)
	for _, title := range []string{"Page One", "Page Two", "Page Three"} {
		createFn(context.Background(), &pluginv1.ToolRequest{
			Arguments: makeArgs(t, map[string]any{
				"project_id": "multi-proj",
				"title":      title,
				"body":       "Content",
			}),
		})
	}

	listFn := tools.DocList(store)
	resp, err := listFn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{"project_id": "multi-proj"}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSuccess(t, resp)
	text := responseText(t, resp)
	if !strings.Contains(text, "page-one") {
		t.Fatalf("expected page-one in list, got: %s", text)
	}
	if !strings.Contains(text, "page-two") {
		t.Fatalf("expected page-two in list, got: %s", text)
	}
}

func TestDocList_CategoryFilter(t *testing.T) {
	store := makeStore()
	createFn := tools.DocCreate(store)
	createFn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"title":      "Arch Doc",
			"body":       "content",
			"category":   "architecture",
		}),
	})
	createFn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"title":      "Guide Doc",
			"body":       "content",
			"category":   "guides",
		}),
	})

	listFn := tools.DocList(store)
	resp, err := listFn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"category":   "architecture",
		}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSuccess(t, resp)
	text := responseText(t, resp)
	if !strings.Contains(text, "arch-doc") {
		t.Fatalf("expected arch-doc in filtered list, got: %s", text)
	}
	if strings.Contains(text, "guide-doc") {
		t.Fatalf("expected guide-doc filtered OUT, got: %s", text)
	}
}

// ---------- doc_delete ----------

func TestDocDelete_Succeeds(t *testing.T) {
	store := makeStore()
	// Create
	tools.DocCreate(store)(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"title":      "Delete This",
			"body":       "gone",
		}),
	})

	fn := tools.DocDelete(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"slug":       "delete-this",
		}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSuccess(t, resp)
}

func TestDocDelete_MissingSlug(t *testing.T) {
	store := makeStore()
	fn := tools.DocDelete(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{"project_id": "proj"}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertError(t, resp, "validation_error")
}

// ---------- doc_search ----------

func TestDocSearch_FindsMatch(t *testing.T) {
	store := makeStore()
	tools.DocCreate(store)(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"title":      "Deployment Guide",
			"body":       "How to deploy to production using Docker.",
		}),
	})

	fn := tools.DocSearch(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"query":      "docker",
		}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSuccess(t, resp)
	text := responseText(t, resp)
	if !strings.Contains(text, "deployment-guide") {
		t.Fatalf("expected deployment-guide in search results, got: %s", text)
	}
}

func TestDocSearch_NoMatch(t *testing.T) {
	store := makeStore()
	fn := tools.DocSearch(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{
			"project_id": "proj",
			"query":      "nonexistent-term-xyz",
		}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSuccess(t, resp)
	text := responseText(t, resp)
	if !strings.Contains(text, "No docs found") {
		t.Fatalf("expected no results message, got: %s", text)
	}
}

func TestDocSearch_MissingQuery(t *testing.T) {
	store := makeStore()
	fn := tools.DocSearch(store)
	resp, err := fn(context.Background(), &pluginv1.ToolRequest{
		Arguments: makeArgs(t, map[string]any{"project_id": "proj"}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertError(t, resp, "validation_error")
}
