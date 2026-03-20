package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps a Neo4j driver and provides typed helpers for Cypher execution.
type Client struct {
	driver neo4j.DriverWithContext
}

// NewClient creates and verifies a Neo4j client connection.
func NewClient(ctx context.Context, uri, user, password string) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		return nil, fmt.Errorf("create neo4j driver: %w", err)
	}
	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, fmt.Errorf("verify neo4j connectivity: %w", err)
	}
	return &Client{driver: driver}, nil
}

// Close releases the underlying driver resources.
func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// RunCypher executes a write Cypher statement inside an explicit transaction.
func (c *Client) RunCypher(ctx context.Context, query string, params map[string]any) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	_, err := neo4j.ExecuteWrite(ctx, session, func(tx neo4j.ManagedTransaction) (struct{}, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return struct{}{}, err
		}
		// Consume result to allow the transaction to complete.
		_, err = result.Consume(ctx)
		return struct{}{}, err
	})
	return err
}

// QueryNodes runs a read Cypher query and returns all records as key→value maps.
func (c *Client) QueryNodes(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var records []map[string]any
	for result.Next(ctx) {
		record := result.Record()
		row := make(map[string]any, len(record.Keys))
		for _, key := range record.Keys {
			row[key], _ = record.Get(key)
		}
		records = append(records, row)
	}
	return records, result.Err()
}
