package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

// ensureQdrantCollection checks service availability and ensures the collection exists.
func ensureQdrantCollection(ctx context.Context, baseURL, collection string, dim int, apiKey string) error {
	if baseURL == "" || collection == "" || dim <= 0 {
		return fmt.Errorf("invalid qdrant params")
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("parse qdrant url: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Check if collection exists
	getURL := *u
	getURL.Path = path.Join(getURL.Path, "/collections", collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getURL.String(), nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("api-key", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Parse vectors size and verify dimension
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
			if sz, ok := extractQdrantVectorSize(body); ok && sz > 0 {
				if sz != dim {
					return fmt.Errorf("collection '%s' dimension mismatch: existing=%d, current=%d. Fix: use a new collection via --collection, delete the old collection in Qdrant, or keep the same embedding model", collection, sz, dim)
				}
			}
		}
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("check collection failed: %s", resp.Status)
	}

	// Create collection with cosine distance
	payload := map[string]any{
		"vectors": map[string]any{
			"size":     dim,
			"distance": "Cosine",
		},
	}
	b, _ := json.Marshal(payload)

	putURL := *u
	putURL.Path = path.Join(putURL.Path, "/collections", collection)
	preq, err := http.NewRequestWithContext(ctx, http.MethodPut, putURL.String(), bytes.NewReader(b))
	if err != nil {
		return err
	}
	preq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		preq.Header.Set("api-key", apiKey)
	}
	pr, err := client.Do(preq)
	if err != nil {
		return fmt.Errorf("create collection request failed: %w", err)
	}
	defer pr.Body.Close()
	if pr.StatusCode != http.StatusOK {
		return fmt.Errorf("create collection failed: %s", pr.Status)
	}
	return nil
}

// extractQdrantVectorSize tries to read vectors.size from GET /collections/{name} response.
func extractQdrantVectorSize(body map[string]any) (int, bool) {
	res, ok := body["result"].(map[string]any)
	if !ok {
		return 0, false
	}
	cfg, ok := res["config"].(map[string]any)
	if !ok {
		return 0, false
	}
	params, ok := cfg["params"].(map[string]any)
	if !ok {
		return 0, false
	}
	vectors, ok := params["vectors"].(map[string]any)
	if !ok {
		return 0, false
	}
	// Case 1: single vector { size, distance }
	if v, ok := vectors["size"]; ok {
		if f, ok := v.(float64); ok {
			return int(f), true
		}
	}
	// Case 2: named vectors map: { name: { size, distance}, ... }
	for _, vv := range vectors {
		if mv, ok := vv.(map[string]any); ok {
			if v, ok := mv["size"]; ok {
				if f, ok := v.(float64); ok {
					return int(f), true
				}
			}
		}
	}
	return 0, false
}
