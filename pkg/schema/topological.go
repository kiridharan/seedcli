// Package schema provides topological sorting for dependency resolution
package schema

import (
	"fmt"

	"github.com/kiridharan/seedcli/pkg/core"
)

// TopologicalSort sorts collections by foreign key dependencies
// Returns collections in order where dependencies come first
func TopologicalSort(collections []*core.Collection) ([]*core.Collection, error) {
	// Build adjacency list and in-degree map
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	collectionMap := make(map[string]*core.Collection)

	// Initialize
	for _, col := range collections {
		collectionMap[col.Name] = col
		graph[col.Name] = []string{}
		inDegree[col.Name] = 0
	}

	// Build edges based on foreign keys
	// Edge from A to B means A depends on B (A references B)
	for _, col := range collections {
		for _, fk := range col.ForeignKeys {
			// Skip self-references
			if fk.ReferencedTable == col.Name {
				continue
			}

			// Only add edge if referenced table is in our collection set
			if _, exists := collectionMap[fk.ReferencedTable]; exists {
				graph[fk.ReferencedTable] = append(graph[fk.ReferencedTable], col.Name)
				inDegree[col.Name]++
			}
		}
	}

	// Kahn's algorithm
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var result []*core.Collection
	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]

		result = append(result, collectionMap[current])

		// Process neighbors
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(result) != len(collections) {
		return nil, fmt.Errorf("cyclic dependency detected")
	}

	return result, nil
}

// GetSelfReferences returns a map of table names to their self-referencing foreign keys
func GetSelfReferences(collections []*core.Collection) map[string][]*core.ForeignKey {
	result := make(map[string][]*core.ForeignKey)

	for _, col := range collections {
		var selfRefs []*core.ForeignKey
		for _, fk := range col.ForeignKeys {
			if fk.ReferencedTable == col.Name {
				selfRefs = append(selfRefs, fk)
			}
		}
		if len(selfRefs) > 0 {
			result[col.Name] = selfRefs
		}
	}

	return result
}

// GetDependencyGraph returns the dependency graph as an adjacency list
// Key is table name, value is list of tables it depends on
func GetDependencyGraph(collections []*core.Collection) map[string][]string {
	graph := make(map[string][]string)

	for _, col := range collections {
		deps := []string{}
		for _, fk := range col.ForeignKeys {
			if fk.ReferencedTable != col.Name {
				deps = append(deps, fk.ReferencedTable)
			}
		}
		graph[col.Name] = deps
	}

	return graph
}

// DetectCycles detects if there are any cycles in the dependency graph
func DetectCycles(collections []*core.Collection) [][]string {
	graph := GetDependencyGraph(collections)
	visited := make(map[string]int) // 0: unvisited, 1: in progress, 2: completed
	var cycles [][]string

	var dfs func(node string, path []string) bool
	dfs = func(node string, path []string) bool {
		if visited[node] == 1 {
			// Found cycle - find where it starts
			cycleStart := -1
			for i, n := range path {
				if n == node {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := append(path[cycleStart:], node)
				cycles = append(cycles, cycle)
			}
			return true
		}

		if visited[node] == 2 {
			return false
		}

		visited[node] = 1
		path = append(path, node)

		for _, dep := range graph[node] {
			dfs(dep, path)
		}

		visited[node] = 2
		return false
	}

	for _, col := range collections {
		if visited[col.Name] == 0 {
			dfs(col.Name, []string{})
		}
	}

	return cycles
}

// GetInsertionOrder returns the order to insert data considering dependencies
// This is similar to TopologicalSort but with additional handling for cycles
func GetInsertionOrder(collections []*core.Collection) []*core.Collection {
	sorted, err := TopologicalSort(collections)
	if err != nil {
		// If there are cycles, return original order with self-refs last
		return reorderWithSelfRefsLast(collections)
	}
	return sorted
}

// reorderWithSelfRefsLast moves tables with self-references to the end
func reorderWithSelfRefsLast(collections []*core.Collection) []*core.Collection {
	selfRefs := GetSelfReferences(collections)

	var withoutSelfRef []*core.Collection
	var withSelfRef []*core.Collection

	for _, col := range collections {
		if _, hasSelfRef := selfRefs[col.Name]; hasSelfRef {
			withSelfRef = append(withSelfRef, col)
		} else {
			withoutSelfRef = append(withoutSelfRef, col)
		}
	}

	return append(withoutSelfRef, withSelfRef...)
}

// DependencyInfo provides detailed dependency information for a collection
type DependencyInfo struct {
	Collection    string
	DependsOn     []string
	DependedBy    []string
	SelfReference bool
	Depth         int
}

// AnalyzeDependencies provides detailed dependency analysis
func AnalyzeDependencies(collections []*core.Collection) map[string]*DependencyInfo {
	result := make(map[string]*DependencyInfo)
	selfRefs := GetSelfReferences(collections)

	// Initialize
	for _, col := range collections {
		result[col.Name] = &DependencyInfo{
			Collection:    col.Name,
			DependsOn:     []string{},
			DependedBy:    []string{},
			SelfReference: len(selfRefs[col.Name]) > 0,
		}
	}

	// Build dependencies
	for _, col := range collections {
		for _, fk := range col.ForeignKeys {
			if fk.ReferencedTable != col.Name {
				result[col.Name].DependsOn = append(result[col.Name].DependsOn, fk.ReferencedTable)
				if info, exists := result[fk.ReferencedTable]; exists {
					info.DependedBy = append(info.DependedBy, col.Name)
				}
			}
		}
	}

	// Calculate depth
	calculateDepths(collections, result)

	return result
}

// calculateDepths calculates the dependency depth for each collection
func calculateDepths(collections []*core.Collection, info map[string]*DependencyInfo) {
	depths := make(map[string]int)

	var getDepth func(name string, visited map[string]bool) int
	getDepth = func(name string, visited map[string]bool) int {
		if d, ok := depths[name]; ok {
			return d
		}

		if visited[name] {
			return 0 // Cycle
		}
		visited[name] = true

		maxDepth := 0
		if i, ok := info[name]; ok {
			for _, dep := range i.DependsOn {
				d := getDepth(dep, visited)
				if d+1 > maxDepth {
					maxDepth = d + 1
				}
			}
		}

		depths[name] = maxDepth
		return maxDepth
	}

	for _, col := range collections {
		visited := make(map[string]bool)
		info[col.Name].Depth = getDepth(col.Name, visited)
	}
}
