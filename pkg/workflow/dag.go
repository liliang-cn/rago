package workflow

import (
	"fmt"
	"sync"
	"time"
)

// DAG represents a Directed Acyclic Graph for workflow execution
type DAG struct {
	nodes    map[string]*DAGNode
	edges    map[string][]string // node -> list of nodes it points to
	inDegree map[string]int      // node -> in-degree count
	mu       sync.RWMutex
}

// DAGNode represents a node in the DAG
type DAGNode struct {
	ID           string
	Data         interface{}
	Level        int      // Execution level for parallel processing
	Dependencies []string // Nodes this node depends on
	Dependents   []string // Nodes that depend on this node
}

// NewDAG creates a new DAG
func NewDAG() *DAG {
	return &DAG{
		nodes:    make(map[string]*DAGNode),
		edges:    make(map[string][]string),
		inDegree: make(map[string]int),
	}
}

// AddNode adds a node to the DAG
func (d *DAG) AddNode(id string, data interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.nodes[id] = &DAGNode{
		ID:           id,
		Data:         data,
		Level:        -1,
		Dependencies: []string{},
		Dependents:   []string{},
	}
	d.inDegree[id] = 0
}

// AddEdge adds a directed edge from source to target
func (d *DAG) AddEdge(source, target string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if nodes exist
	if _, exists := d.nodes[source]; !exists {
		return fmt.Errorf("source node not found: %s", source)
	}
	if _, exists := d.nodes[target]; !exists {
		return fmt.Errorf("target node not found: %s", target)
	}

	// Add edge
	if d.edges[source] == nil {
		d.edges[source] = []string{}
	}
	d.edges[source] = append(d.edges[source], target)

	// Update dependencies and dependents
	d.nodes[target].Dependencies = append(d.nodes[target].Dependencies, source)
	d.nodes[source].Dependents = append(d.nodes[source].Dependents, target)
	d.inDegree[target]++

	return nil
}

// HasCycle checks if the DAG contains a cycle
func (d *DAG) HasCycle() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for nodeID := range d.nodes {
		if !visited[nodeID] {
			if d.hasCycleDFS(nodeID, visited, recStack) {
				return true
			}
		}
	}

	return false
}

// hasCycleDFS performs DFS to detect cycles
func (d *DAG) hasCycleDFS(nodeID string, visited, recStack map[string]bool) bool {
	visited[nodeID] = true
	recStack[nodeID] = true

	// Check all adjacent nodes
	if edges, exists := d.edges[nodeID]; exists {
		for _, neighbor := range edges {
			if !visited[neighbor] {
				if d.hasCycleDFS(neighbor, visited, recStack) {
					return true
				}
			} else if recStack[neighbor] {
				// Found a back edge - cycle detected
				return true
			}
		}
	}

	recStack[nodeID] = false
	return false
}

// TopologicalSort returns nodes in topologically sorted order
func (d *DAG) TopologicalSort() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.HasCycle() {
		return nil, fmt.Errorf("cannot perform topological sort on a graph with cycles")
	}

	visited := make(map[string]bool)
	stack := []string{}

	for nodeID := range d.nodes {
		if !visited[nodeID] {
			d.topologicalSortDFS(nodeID, visited, &stack)
		}
	}

	// Reverse the stack to get the correct order
	result := make([]string, len(stack))
	for i, j := 0, len(stack)-1; j >= 0; i, j = i+1, j-1 {
		result[i] = stack[j]
	}

	return result, nil
}

// topologicalSortDFS performs DFS for topological sorting
func (d *DAG) topologicalSortDFS(nodeID string, visited map[string]bool, stack *[]string) {
	visited[nodeID] = true

	if edges, exists := d.edges[nodeID]; exists {
		for _, neighbor := range edges {
			if !visited[neighbor] {
				d.topologicalSortDFS(neighbor, visited, stack)
			}
		}
	}

	*stack = append(*stack, nodeID)
}

// GetExecutionPlan returns execution levels for parallel processing
func (d *DAG) GetExecutionPlan() [][]string {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Calculate levels for each node
	d.calculateLevels()

	// Group nodes by level
	levelMap := make(map[int][]string)
	maxLevel := -1

	for nodeID, node := range d.nodes {
		if node.Level >= 0 {
			if levelMap[node.Level] == nil {
				levelMap[node.Level] = []string{}
			}
			levelMap[node.Level] = append(levelMap[node.Level], nodeID)
			if node.Level > maxLevel {
				maxLevel = node.Level
			}
		}
	}

	// Convert to ordered slice
	plan := make([][]string, maxLevel+1)
	for level := 0; level <= maxLevel; level++ {
		plan[level] = levelMap[level]
	}

	return plan
}

// calculateLevels calculates execution levels for each node
func (d *DAG) calculateLevels() {
	// Reset all levels
	for _, node := range d.nodes {
		node.Level = -1
	}

	// Find nodes with no dependencies (level 0)
	queue := []string{}
	for nodeID, node := range d.nodes {
		if len(node.Dependencies) == 0 {
			node.Level = 0
			queue = append(queue, nodeID)
		}
	}

	// Process nodes level by level
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Process dependents
		if edges, exists := d.edges[current]; exists {
			for _, dependent := range edges {
				depNode := d.nodes[dependent]

				// Calculate max level from dependencies
				maxLevel := -1
				allDepsProcessed := true

				for _, dep := range depNode.Dependencies {
					depLevel := d.nodes[dep].Level
					if depLevel == -1 {
						allDepsProcessed = false
						break
					}
					if depLevel > maxLevel {
						maxLevel = depLevel
					}
				}

				// If all dependencies are processed and this node hasn't been assigned a level
				if allDepsProcessed && depNode.Level == -1 {
					depNode.Level = maxLevel + 1
					queue = append(queue, dependent)
				}
			}
		}
	}
}

// GetNode returns a node by ID
func (d *DAG) GetNode(id string) (*DAGNode, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, exists := d.nodes[id]
	return node, exists
}

// GetNodes returns all nodes
func (d *DAG) GetNodes() map[string]*DAGNode {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy to prevent external modifications
	nodesCopy := make(map[string]*DAGNode, len(d.nodes))
	for id, node := range d.nodes {
		nodesCopy[id] = &DAGNode{
			ID:           node.ID,
			Data:         node.Data,
			Level:        node.Level,
			Dependencies: append([]string{}, node.Dependencies...),
			Dependents:   append([]string{}, node.Dependents...),
		}
	}

	return nodesCopy
}

// GetEdges returns all edges
func (d *DAG) GetEdges() map[string][]string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy
	edgesCopy := make(map[string][]string, len(d.edges))
	for source, targets := range d.edges {
		edgesCopy[source] = append([]string{}, targets...)
	}

	return edgesCopy
}

// FindPath finds a path from source to target node
func (d *DAG) FindPath(source, target string) ([]string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if source == target {
		return []string{source}, true
	}

	visited := make(map[string]bool)
	path := []string{}
	
	if d.findPathDFS(source, target, visited, &path) {
		return path, true
	}

	return nil, false
}

// findPathDFS performs DFS to find a path
func (d *DAG) findPathDFS(current, target string, visited map[string]bool, path *[]string) bool {
	visited[current] = true
	*path = append(*path, current)

	if current == target {
		return true
	}

	if edges, exists := d.edges[current]; exists {
		for _, neighbor := range edges {
			if !visited[neighbor] {
				if d.findPathDFS(neighbor, target, visited, path) {
					return true
				}
			}
		}
	}

	// Backtrack
	*path = (*path)[:len(*path)-1]
	return false
}

// GetParallelGroups returns groups of nodes that can be executed in parallel
func (d *DAG) GetParallelGroups() [][]string {
	return d.GetExecutionPlan()
}

// Validate checks if the DAG is valid for workflow execution
func (d *DAG) Validate() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check for cycles
	if d.HasCycle() {
		return fmt.Errorf("DAG contains cycles")
	}

	// Check for disconnected nodes (optional - depends on requirements)
	// A disconnected node might be valid in some workflow scenarios

	return nil
}

// Clone creates a deep copy of the DAG
func (d *DAG) Clone() *DAG {
	d.mu.RLock()
	defer d.mu.RUnlock()

	clone := NewDAG()

	// Clone nodes
	for id, node := range d.nodes {
		clone.nodes[id] = &DAGNode{
			ID:           node.ID,
			Data:         node.Data,
			Level:        node.Level,
			Dependencies: append([]string{}, node.Dependencies...),
			Dependents:   append([]string{}, node.Dependents...),
		}
	}

	// Clone edges
	for source, targets := range d.edges {
		clone.edges[source] = append([]string{}, targets...)
	}

	return clone
}

// Stats returns statistics about the DAG
type DAGStats struct {
	NodeCount      int     `json:"node_count"`
	EdgeCount      int     `json:"edge_count"`
	MaxLevel       int     `json:"max_level"`
	AvgDependencies float64 `json:"avg_dependencies"`
	IsCyclic       bool    `json:"is_cyclic"`
}

// GetDependencies returns the dependencies of a node
func (d *DAG) GetDependencies(nodeID string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if node, exists := d.nodes[nodeID]; exists {
		return append([]string{}, node.Dependencies...)
	}
	return []string{}
}

// GetDependents returns the dependents of a node
func (d *DAG) GetDependents(nodeID string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if node, exists := d.nodes[nodeID]; exists {
		return append([]string{}, node.Dependents...)
	}
	return []string{}
}

// GetRoots returns all root nodes (nodes with no dependencies)
func (d *DAG) GetRoots() []*DAGNode {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var roots []*DAGNode
	for _, node := range d.nodes {
		if len(node.Dependencies) == 0 {
			roots = append(roots, node)
		}
	}
	return roots
}

// GetLeaves returns all leaf nodes (nodes with no dependents)
func (d *DAG) GetLeaves() []*DAGNode {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var leaves []*DAGNode
	for _, node := range d.nodes {
		if len(node.Dependents) == 0 {
			leaves = append(leaves, node)
		}
	}
	return leaves
}

// ValidateWorkflow validates a workflow for DAG constraints
func (d *DAG) ValidateWorkflow(workflow *Workflow) error {
	if len(workflow.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	// Check for duplicate step IDs
	stepIDs := make(map[string]bool)
	for _, step := range workflow.Steps {
		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepIDs[step.ID] = true
	}

	// Check for invalid dependencies
	for _, step := range workflow.Steps {
		for _, dep := range step.Dependencies {
			if !stepIDs[dep] {
				return fmt.Errorf("dependency step not found: %s", dep)
			}
			if dep == step.ID {
				return fmt.Errorf("step %s has self-dependency", step.ID)
			}
		}
	}

	// Build DAG and check for cycles
	tempDAG := NewDAG()
	for _, step := range workflow.Steps {
		tempDAG.AddNode(step.ID, &step)
	}
	for _, step := range workflow.Steps {
		for _, dep := range step.Dependencies {
			tempDAG.AddEdge(dep, step.ID)
		}
	}

	if tempDAG.HasCycle() {
		return fmt.Errorf("cycle detected in workflow")
	}

	return nil
}

// ExecutionPlan represents an optimized execution plan
type ExecutionPlan struct {
	Levels           [][]*Step     `json:"levels"`
	MaxParallelism   int           `json:"max_parallelism"`
	EstimatedDuration time.Duration `json:"estimated_duration"`
}

// OptimizeExecution creates an optimized execution plan
func (d *DAG) OptimizeExecution(workflow *Workflow) (*ExecutionPlan, error) {
	// Validate workflow first
	if err := d.ValidateWorkflow(workflow); err != nil {
		return nil, err
	}

	// Build DAG
	tempDAG := NewDAG()
	for _, step := range workflow.Steps {
		tempDAG.AddNode(step.ID, &step)
	}
	for _, step := range workflow.Steps {
		for _, dep := range step.Dependencies {
			tempDAG.AddEdge(dep, step.ID)
		}
	}

	// Get parallel groups
	parallelGroups := tempDAG.GetExecutionPlan()

	// Convert to ExecutionPlan
	plan := &ExecutionPlan{
		Levels: make([][]*Step, len(parallelGroups)),
	}

	maxParallel := 0
	for i, group := range parallelGroups {
		plan.Levels[i] = make([]*Step, len(group))
		for j, stepID := range group {
			for _, step := range workflow.Steps {
				if step.ID == stepID {
					plan.Levels[i][j] = &step
					break
				}
			}
		}
		if len(group) > maxParallel {
			maxParallel = len(group)
		}
	}

	plan.MaxParallelism = maxParallel
	plan.EstimatedDuration = time.Duration(len(parallelGroups)) * time.Minute // Rough estimate

	return plan, nil
}

// GetStats returns statistics about the DAG
func (d *DAG) GetStats() *DAGStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := &DAGStats{
		NodeCount: len(d.nodes),
		IsCyclic:  d.HasCycle(),
	}

	// Count edges and dependencies
	totalDeps := 0
	maxLevel := -1
	
	for _, node := range d.nodes {
		totalDeps += len(node.Dependencies)
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}

	for _, targets := range d.edges {
		stats.EdgeCount += len(targets)
	}

	if stats.NodeCount > 0 {
		stats.AvgDependencies = float64(totalDeps) / float64(stats.NodeCount)
	}

	stats.MaxLevel = maxLevel

	return stats
}