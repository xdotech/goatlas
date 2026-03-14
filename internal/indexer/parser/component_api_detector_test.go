package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectComponentAPICalls_ConstantPaths(t *testing.T) {
	// Create a temp file that mimics the real PMC dashboard pattern
	content := `
import { api } from '@services/apiClient';
import { MasterDataApi } from './masterData.api';

// Pattern 1: Constant reference (no string literal)
export const useReasons = (params: any) => ({
    queryFn: () => api.get(MasterDataApi.REASON, { params })
});

// Pattern 2: Template literal with constant + suffix
export const useDeleteReason = () => ({
    mutationFn: (uid: string) => api.delete(` + "`${MasterDataApi.REASON}/${uid}`" + `)
});

// Pattern 3: generatePath with constant
export const useItemDetail = (uid: string) => ({
    queryFn: () => api.get(generatePath(MasterDataApi.ITEM_DETAIL, { uid })),
});

// Pattern 4: Traditional literal path
export const useLegacy = () => ({
    queryFn: () => api.get('/api/v1/legacy-items', { params: {} })
});

// Pattern 5: post with constant
export const useCreateItem = () => ({
    mutationFn: (data: any) => api.post(MasterDataApi.ITEM, data)
});

// Pattern 6: put with template literal
export const useUpdateItem = () => ({
    mutationFn: ({ id, data }: any) => api.put(` + "`${MasterDataApi.ITEM}/${id}`" + `, data)
});
`

	dir := t.TempDir()
	fp := filepath.Join(dir, "test.query.ts")
	if err := os.WriteFile(fp, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	calls, err := DetectComponentAPICalls(fp, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Found %d API calls:", len(calls))
	for i, c := range calls {
		t.Logf("  [%d] Component=%-20s Method=%-6s Path=%s (line %d)", i+1, c.Component, c.HttpMethod, c.APIPath, c.Line)
	}

	if len(calls) < 6 {
		t.Errorf("expected at least 6 API calls, got %d", len(calls))
	}

	// Verify specific patterns were detected
	found := map[string]bool{}
	for _, c := range calls {
		key := c.HttpMethod + ":" + c.APIPath
		found[key] = true
	}

	// Pattern 1: constant reference
	if !found["GET:MasterDataApi.REASON"] {
		t.Error("missing: GET MasterDataApi.REASON (constant reference)")
	}
	// Pattern 2: template literal with constant
	if !found["DELETE:MasterDataApi.REASON/${uid}"] {
		t.Error("missing: DELETE MasterDataApi.REASON/${uid} (template const path)")
	}
	// Pattern 3: generatePath
	if !found["GET:MasterDataApi.ITEM_DETAIL"] {
		t.Error("missing: GET MasterDataApi.ITEM_DETAIL (generatePath)")
	}
	// Pattern 4: literal path
	if !found["GET:/api/v1/legacy-items"] {
		t.Error("missing: GET /api/v1/legacy-items (literal path)")
	}
	// Pattern 5: post with constant
	if !found["POST:MasterDataApi.ITEM"] {
		t.Error("missing: POST MasterDataApi.ITEM (constant reference)")
	}
	// Pattern 6: put with template literal
	if !found["PUT:MasterDataApi.ITEM/${id}"] {
		t.Error("missing: PUT MasterDataApi.ITEM/${id} (template const path)")
	}
}

func TestDetectComponentAPICalls_RealFile(t *testing.T) {
	// Test with a real file from pmc-wms-dashboard if available
	realFile := "/Users/xuando/Projects/pmc-wms/pmc-wms-dashboard/src/features/masterData/api/masterData.query.ts"
	if _, err := os.Stat(realFile); os.IsNotExist(err) {
		t.Skip("pmc-wms-dashboard not available, skipping real file test")
	}

	calls, err := DetectComponentAPICalls(realFile, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Found %d API calls from real file", len(calls))
	for i, c := range calls {
		t.Logf("  [%d] Component=%-20s Method=%-6s Path=%s (line %d)", i+1, c.Component, c.HttpMethod, c.APIPath, c.Line)
	}

	if len(calls) == 0 {
		t.Error("expected API calls from real file, got 0")
	}
}
