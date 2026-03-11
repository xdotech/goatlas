package vector

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/qdrant/go-client/qdrant"
)

const collectionName = "code_symbols"
const vectorSize = 768

// QdrantClient wraps the high-level Qdrant client.
type QdrantClient struct {
	client *qdrant.Client
}

// NewQdrantClient connects to Qdrant at the given URL (HTTP or bare host:port).
// gRPC default port is 6334.
func NewQdrantClient(ctx context.Context, addr string) (*QdrantClient, error) {
	host, port := parseQdrantAddr(addr)
	cfg := &qdrant.Config{
		Host:                   host,
		Port:                   port,
		UseTLS:                 false,
		SkipCompatibilityCheck: true,
	}
	c, err := qdrant.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect qdrant %s:%d: %w", host, port, err)
	}
	return &QdrantClient{client: c}, nil
}

// parseQdrantAddr extracts host and port from HTTP URL or host:port string.
func parseQdrantAddr(addr string) (string, int) {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		u, err := url.Parse(addr)
		if err == nil {
			port := 6334
			if p := u.Port(); p != "" {
				fmt.Sscanf(p, "%d", &port)
			}
			return u.Hostname(), port
		}
	}
	// Bare host:port
	host := addr
	port := 6334
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		host = addr[:idx]
		fmt.Sscanf(addr[idx+1:], "%d", &port)
	}
	return host, port
}

// Close releases the Qdrant connection.
func (c *QdrantClient) Close() error {
	return c.client.Close()
}

// EnsureCollection creates the collection if it does not exist.
func (c *QdrantClient) EnsureCollection(ctx context.Context) error {
	names, err := c.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("list collections: %w", err)
	}
	for _, n := range names {
		if n == collectionName {
			return nil
		}
	}
	size := uint64(vectorSize)
	return c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     size,
			Distance: qdrant.Distance_Cosine,
		}),
	})
}

// UpsertPoints inserts or updates the given CodePoints in Qdrant.
func (c *QdrantClient) UpsertPoints(ctx context.Context, points []CodePoint) error {
	if len(points) == 0 {
		return nil
	}
	qPoints := make([]*qdrant.PointStruct, len(points))
	for i, p := range points {
		payload := qdrant.NewValueMap(p.Payload)
		qPoints[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(p.ID),
			Vectors: qdrant.NewVectors(p.Vector...),
			Payload: payload,
		}
	}
	_, err := c.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         qPoints,
	})
	return err
}

// Search performs a nearest-neighbor search and returns matching results.
func (c *QdrantClient) Search(ctx context.Context, vector []float32, limit int, filter map[string]string) ([]SearchResult, error) {
	var qFilter *qdrant.Filter
	if len(filter) > 0 {
		conditions := make([]*qdrant.Condition, 0, len(filter))
		for k, v := range filter {
			conditions = append(conditions, qdrant.NewMatch(k, v))
		}
		qFilter = &qdrant.Filter{Must: conditions}
	}

	scored, err := c.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQueryDense(vector),
		Limit:          uintPtr(uint64(limit)),
		Filter:         qFilter,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant query: %w", err)
	}

	results := make([]SearchResult, 0, len(scored))
	for _, hit := range scored {
		r := SearchResult{Score: hit.GetScore()}
		pl := hit.GetPayload()
		if v, ok := pl["symbol_id"]; ok {
			r.SymbolID = v.GetIntegerValue()
		}
		if v, ok := pl["file"]; ok {
			r.File = v.GetStringValue()
		}
		if v, ok := pl["kind"]; ok {
			r.Kind = v.GetStringValue()
		}
		if v, ok := pl["name"]; ok {
			r.Name = v.GetStringValue()
		}
		if v, ok := pl["service"]; ok {
			r.Service = v.GetStringValue()
		}
		if v, ok := pl["line"]; ok {
			r.Line = int(v.GetIntegerValue())
		}
		results = append(results, r)
	}
	return results, nil
}

func uintPtr(v uint64) *uint64 { return &v }
