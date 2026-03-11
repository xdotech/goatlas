package vector

// CodePoint represents a symbol embedded in vector space.
type CodePoint struct {
	ID      uint64
	Vector  []float32
	Payload map[string]interface{}
}

// SearchResult from vector search.
type SearchResult struct {
	SymbolID int64
	Score    float32
	File     string
	Kind     string
	Name     string
	Line     int
	Service  string
}
