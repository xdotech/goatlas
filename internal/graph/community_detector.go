package graph

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CommunityDetector implements Louvain community detection on the call graph.
type CommunityDetector struct {
	pool          *pgxpool.Pool
	communityRepo domain.CommunityRepository
}

// NewCommunityDetector constructs a CommunityDetector.
func NewCommunityDetector(pool *pgxpool.Pool, communityRepo domain.CommunityRepository) *CommunityDetector {
	return &CommunityDetector{pool: pool, communityRepo: communityRepo}
}

// Edge represents a directed edge in the call graph.
type Edge struct {
	From   int
	To     int
	Weight float64
}

// DetectAll runs Louvain community detection for a repository and persists results.
func (d *CommunityDetector) DetectAll(ctx context.Context, repoID int64) (int, error) {
	// Clear previous results
	if err := d.communityRepo.DeleteByRepoID(ctx, repoID); err != nil {
		return 0, fmt.Errorf("clear old communities: %w", err)
	}

	nodes, nodeFiles, edges, err := d.BuildGraph(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("build graph: %w", err)
	}
	if len(nodes) == 0 {
		return 0, nil
	}

	assignments := d.RunLouvain(nodes, edges)

	// Group nodes by community
	groups := map[int][]string{}
	for node, commID := range assignments {
		groups[commID] = append(groups[commID], node)
	}

	count := 0
	for commID, members := range groups {
		if len(members) < 2 {
			continue // skip singleton communities
		}

		name := deriveCommunityName(members)
		comm := &domain.Community{
			RepoID:      repoID,
			CommunityID: commID,
			Name:        name,
			MemberCount: len(members),
		}
		dbID, err := d.communityRepo.Insert(ctx, comm)
		if err != nil {
			continue
		}

		domainMembers := make([]domain.CommunityMember, len(members))
		for i, m := range members {
			domainMembers[i] = domain.CommunityMember{
				CommunityID: dbID,
				SymbolName:  m,
				FilePath:    nodeFiles[m],
			}
		}
		if err := d.communityRepo.InsertMembers(ctx, domainMembers); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// BuildGraph constructs the adjacency list from function_calls table.
func (d *CommunityDetector) BuildGraph(ctx context.Context, repoID int64) ([]string, map[string]string, []Edge, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT fc.caller_qualified_name, fc.callee_name, f.path
		FROM function_calls fc
		JOIN files f ON fc.file_id = f.id
		WHERE f.repo_id = $1
	`, repoID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	nodeIndex := map[string]int{}
	nodeFiles := map[string]string{}
	var nodes []string
	var rawEdges []struct{ from, to string }

	for rows.Next() {
		var caller, callee, filePath string
		if err := rows.Scan(&caller, &callee, &filePath); err != nil {
			return nil, nil, nil, err
		}

		if _, ok := nodeIndex[caller]; !ok {
			nodeIndex[caller] = len(nodes)
			nodes = append(nodes, caller)
			nodeFiles[caller] = filePath
		}
		if _, ok := nodeIndex[callee]; !ok {
			nodeIndex[callee] = len(nodes)
			nodes = append(nodes, callee)
		}
		rawEdges = append(rawEdges, struct{ from, to string }{caller, callee})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	edges := make([]Edge, len(rawEdges))
	for i, re := range rawEdges {
		edges[i] = Edge{
			From:   nodeIndex[re.from],
			To:     nodeIndex[re.to],
			Weight: 1.0,
		}
	}

	return nodes, nodeFiles, edges, nil
}

// RunLouvain implements the Louvain modularity optimization algorithm.
func (d *CommunityDetector) RunLouvain(nodes []string, edges []Edge) map[string]int {
	n := len(nodes)
	if n == 0 {
		return nil
	}

	// Initialize: each node in its own community
	community := make([]int, n)
	for i := range community {
		community[i] = i
	}

	// Build adjacency: neighbor weights for each node
	adj := make([]map[int]float64, n)
	for i := range adj {
		adj[i] = map[int]float64{}
	}
	for _, e := range edges {
		adj[e.From][e.To] += e.Weight
		adj[e.To][e.From] += e.Weight // treat as undirected for Louvain
	}

	// Total edge weight (m)
	totalWeight := 0.0
	for _, e := range edges {
		totalWeight += e.Weight
	}
	if totalWeight == 0 {
		result := make(map[string]int, n)
		for i, name := range nodes {
			result[name] = i
		}
		return result
	}

	// Weighted degree of each node (k_i)
	degree := make([]float64, n)
	for i := range degree {
		for _, w := range adj[i] {
			degree[i] += w
		}
	}

	maxIterations := 100
	for iter := 0; iter < maxIterations; iter++ {
		improved := false

		for i := 0; i < n; i++ {
			bestComm := community[i]
			bestDelta := 0.0

			// Sum of weights inside current community connected to i
			commWeights := map[int]float64{}
			commTotalDegree := map[int]float64{}
			for j := 0; j < n; j++ {
				if j == i {
					continue
				}
				c := community[j]
				commTotalDegree[c] += degree[j]
				if w, ok := adj[i][j]; ok {
					commWeights[c] += w
				}
			}

			// Try moving i to each neighboring community
			currentComm := community[i]
			for c, kiIn := range commWeights {
				if c == currentComm {
					continue
				}
				sigmaTot := commTotalDegree[c]
				ki := degree[i]

				// Delta Q = [k_{i,in} / m - sigma_tot * k_i / (2m^2)]
				deltaQ := kiIn/totalWeight - sigmaTot*ki/(2.0*totalWeight*totalWeight)

				// Also compute the loss from removing from current community
				kiInCurrent := commWeights[currentComm]
				sigmaTotCurrent := commTotalDegree[currentComm]
				deltaRemove := -(kiInCurrent/totalWeight - sigmaTotCurrent*ki/(2.0*totalWeight*totalWeight))

				netDelta := deltaQ + deltaRemove
				if netDelta > bestDelta {
					bestDelta = netDelta
					bestComm = c
				}
			}

			if bestComm != community[i] && bestDelta > 1e-10 {
				community[i] = bestComm
				improved = true
			}
		}

		if !improved {
			break
		}
	}

	// Compact community IDs
	commMap := map[int]int{}
	nextID := 0
	result := make(map[string]int, n)
	for i, name := range nodes {
		c := community[i]
		if _, ok := commMap[c]; !ok {
			commMap[c] = nextID
			nextID++
		}
		result[name] = commMap[c]
	}

	return result
}

// modularity computes the modularity Q for a given community assignment.
// Used for debugging/validation, not in the hot path.
func modularity(n int, edges []Edge, community []int) float64 {
	totalWeight := 0.0
	for _, e := range edges {
		totalWeight += e.Weight
	}
	if totalWeight == 0 {
		return 0
	}

	degree := make([]float64, n)
	for _, e := range edges {
		degree[e.From] += e.Weight
		degree[e.To] += e.Weight
	}

	q := 0.0
	for _, e := range edges {
		if community[e.From] == community[e.To] {
			q += e.Weight - degree[e.From]*degree[e.To]/(2.0*totalWeight)
		}
	}
	return q / totalWeight
}

// Ensure modularity function is referenced to avoid unused warnings.
var _ = modularity
var _ = math.Abs

// deriveCommunityName generates a name for a community based on its members.
func deriveCommunityName(members []string) string {
	// Find most common package prefix
	pkgCounts := map[string]int{}
	for _, m := range members {
		parts := strings.Split(m, ".")
		if len(parts) >= 2 {
			pkg := parts[0]
			// For qualified names like "handler.(*UserHandler).Create", extract "handler"
			pkg = strings.TrimPrefix(pkg, "(")
			pkg = strings.TrimPrefix(pkg, "*")
			pkgCounts[pkg]++
		}
	}

	if len(pkgCounts) == 0 {
		return fmt.Sprintf("cluster-%d", len(members))
	}

	// Sort by count descending
	type pkgCount struct {
		pkg   string
		count int
	}
	var sorted []pkgCount
	for pkg, count := range pkgCounts {
		sorted = append(sorted, pkgCount{pkg, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	top := sorted[0].pkg
	if len(sorted) > 1 {
		return fmt.Sprintf("%s (+%d packages)", top, len(sorted)-1)
	}
	return top
}
