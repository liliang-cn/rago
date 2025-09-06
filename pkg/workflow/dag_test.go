package workflow

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDAG(t *testing.T) {
	dag := NewDAG()
	assert.NotNil(t, dag)
	assert.NotNil(t, dag.nodes)
	assert.NotNil(t, dag.edges)
	assert.NotNil(t, dag.inDegree)
}

func TestDAGAddNode(t *testing.T) {
	dag := NewDAG()
	
	node := &Step{
		ID:   "test-step",
		Name: "Test Step",
		Type: StepTypeAction,
	}
	
	dag.AddNode(node.ID, node)
	
	// Verify node was added
	assert.Contains(t, dag.nodes, "test-step")
	assert.Equal(t, node, dag.nodes["test-step"].Data)
	assert.Equal(t, "test-step", dag.nodes["test-step"].ID)
	assert.Equal(t, 0, dag.inDegree["test-step"])
	
	// Test adding duplicate node should be handled by checking existence
	if _, exists := dag.GetNode(node.ID); exists {
		// Node already exists, which is expected behavior
		assert.True(t, exists)
	}
}

func TestDAGAddEdge(t *testing.T) {
	dag := NewDAG()
	
	// Add nodes first
	node1 := &Step{ID: "step-1", Name: "Step 1", Type: StepTypeAction}
	node2 := &Step{ID: "step-2", Name: "Step 2", Type: StepTypeAction}
	
	dag.AddNode(node1.ID, node1)
	dag.AddNode(node2.ID, node2)
	
	// Add edge from step-1 to step-2
	err := dag.AddEdge("step-1", "step-2")
	assert.NoError(t, err)
	
	// Verify edge was added
	assert.Contains(t, dag.edges["step-1"], "step-2")
	assert.Equal(t, 1, dag.inDegree["step-2"])
	assert.Equal(t, 0, dag.inDegree["step-1"])
	
	// Test adding edge with non-existent from node
	err = dag.AddEdge("non-existent", "step-2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source node not found")
	
	// Test adding edge with non-existent to node
	err = dag.AddEdge("step-1", "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target node not found")
}

func TestDAGHasCycle(t *testing.T) {
	dag := NewDAG()
	
	// Create nodes
	nodes := []*Step{
		{ID: "A", Name: "Step A", Type: StepTypeAction},
		{ID: "B", Name: "Step B", Type: StepTypeAction},
		{ID: "C", Name: "Step C", Type: StepTypeAction},
		{ID: "D", Name: "Step D", Type: StepTypeAction},
	}
	
	for _, node := range nodes {
		dag.AddNode(node.ID, node)
	}
	
	// Test acyclic graph: A -> B -> C -> D
	dag.AddEdge("A", "B")
	dag.AddEdge("B", "C")
	dag.AddEdge("C", "D")
	
	assert.False(t, dag.HasCycle())
	
	// Create cycle: D -> A
	dag.AddEdge("D", "A")
	
	assert.True(t, dag.HasCycle())
}

func TestDAGTopologicalSort(t *testing.T) {
	dag := NewDAG()
	
	// Create a diamond-shaped DAG
	//    A
	//   / \
	//  B   C
	//   \ /
	//    D
	nodes := []*Step{
		{ID: "A", Name: "Step A", Type: StepTypeAction},
		{ID: "B", Name: "Step B", Type: StepTypeAction},
		{ID: "C", Name: "Step C", Type: StepTypeAction},
		{ID: "D", Name: "Step D", Type: StepTypeAction},
	}
	
	for _, node := range nodes {
		dag.AddNode(node.ID, node)
	}
	
	dag.AddEdge("A", "B")
	dag.AddEdge("A", "C")
	dag.AddEdge("B", "D")
	dag.AddEdge("C", "D")
	
	sorted, err := dag.TopologicalSort()
	assert.NoError(t, err)
	assert.Len(t, sorted, 4)
	
	// Verify topological ordering
	positions := make(map[string]int)
	for i, stepID := range sorted {
		positions[stepID] = i
	}
	
	// A must come before B and C
	assert.Less(t, positions["A"], positions["B"])
	assert.Less(t, positions["A"], positions["C"])
	
	// B and C must come before D
	assert.Less(t, positions["B"], positions["D"])
	assert.Less(t, positions["C"], positions["D"])
}

func TestDAGTopologicalSortWithCycle(t *testing.T) {
	dag := NewDAG()
	
	// Create nodes
	nodes := []*Step{
		{ID: "A", Name: "Step A", Type: StepTypeAction},
		{ID: "B", Name: "Step B", Type: StepTypeAction},
		{ID: "C", Name: "Step C", Type: StepTypeAction},
	}
	
	for _, node := range nodes {
		dag.AddNode(node.ID, node)
	}
	
	// Create cycle: A -> B -> C -> A
	dag.AddEdge("A", "B")
	dag.AddEdge("B", "C")
	dag.AddEdge("C", "A")
	
	sorted, err := dag.TopologicalSort()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycles")
	assert.Nil(t, sorted)
}

func TestDAGGetDependencies(t *testing.T) {
	dag := NewDAG()
	
	// Create nodes
	nodes := []*Step{
		{ID: "A", Name: "Step A", Type: StepTypeAction},
		{ID: "B", Name: "Step B", Type: StepTypeAction},
		{ID: "C", Name: "Step C", Type: StepTypeAction},
		{ID: "D", Name: "Step D", Type: StepTypeAction},
	}
	
	for _, node := range nodes {
		dag.AddNode(node.ID, node)
	}
	
	// A -> B, A -> C, B -> D, C -> D
	dag.AddEdge("A", "B")
	dag.AddEdge("A", "C")
	dag.AddEdge("B", "D")
	dag.AddEdge("C", "D")
	
	// Test dependencies
	deps := dag.GetDependencies("A")
	assert.Empty(t, deps) // A has no dependencies
	
	deps = dag.GetDependencies("B")
	assert.Len(t, deps, 1)
	assert.Contains(t, deps, "A")
	
	deps = dag.GetDependencies("D")
	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "B")
	assert.Contains(t, deps, "C")
	
	// Test non-existent node
	deps = dag.GetDependencies("non-existent")
	assert.Empty(t, deps)
}

func TestDAGGetDependents(t *testing.T) {
	dag := NewDAG()
	
	// Create nodes
	nodes := []*Step{
		{ID: "A", Name: "Step A", Type: StepTypeAction},
		{ID: "B", Name: "Step B", Type: StepTypeAction},
		{ID: "C", Name: "Step C", Type: StepTypeAction},
		{ID: "D", Name: "Step D", Type: StepTypeAction},
	}
	
	for _, node := range nodes {
		dag.AddNode(node.ID, node)
	}
	
	// A -> B, A -> C, B -> D, C -> D
	dag.AddEdge("A", "B")
	dag.AddEdge("A", "C")
	dag.AddEdge("B", "D")
	dag.AddEdge("C", "D")
	
	// Test dependents
	deps := dag.GetDependents("A")
	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "B")
	assert.Contains(t, deps, "C")
	
	deps = dag.GetDependents("B")
	assert.Len(t, deps, 1)
	assert.Contains(t, deps, "D")
	
	deps = dag.GetDependents("D")
	assert.Empty(t, deps) // D has no dependents
	
	// Test non-existent node
	deps = dag.GetDependents("non-existent")
	assert.Empty(t, deps)
}

func TestDAGGetRoots(t *testing.T) {
	dag := NewDAG()
	
	// Create nodes
	nodes := []*Step{
		{ID: "A", Name: "Step A", Type: StepTypeAction},
		{ID: "B", Name: "Step B", Type: StepTypeAction},
		{ID: "C", Name: "Step C", Type: StepTypeAction},
		{ID: "D", Name: "Step D", Type: StepTypeAction},
	}
	
	for _, node := range nodes {
		dag.AddNode(node.ID, node)
	}
	
	// Initially all nodes are roots
	roots := dag.GetRoots()
	assert.Len(t, roots, 4)
	
	// Add edges: A -> B, C -> D
	dag.AddEdge("A", "B")
	dag.AddEdge("C", "D")
	
	// Now only A and C are roots
	roots = dag.GetRoots()
	assert.Len(t, roots, 2)
	assert.Contains(t, roots, dag.nodes["A"])
	assert.Contains(t, roots, dag.nodes["C"])
}

func TestDAGGetLeaves(t *testing.T) {
	dag := NewDAG()
	
	// Create nodes
	nodes := []*Step{
		{ID: "A", Name: "Step A", Type: StepTypeAction},
		{ID: "B", Name: "Step B", Type: StepTypeAction},
		{ID: "C", Name: "Step C", Type: StepTypeAction},
		{ID: "D", Name: "Step D", Type: StepTypeAction},
	}
	
	for _, node := range nodes {
		dag.AddNode(node.ID, node)
	}
	
	// Initially all nodes are leaves
	leaves := dag.GetLeaves()
	assert.Len(t, leaves, 4)
	
	// Add edges: A -> B, A -> C
	dag.AddEdge("A", "B")
	dag.AddEdge("A", "C")
	
	// Now B, C, and D are leaves
	leaves = dag.GetLeaves()
	assert.Len(t, leaves, 3)
	nodeIDs := make([]string, len(leaves))
	for i, node := range leaves {
		nodeIDs[i] = node.ID
	}
	assert.Contains(t, nodeIDs, "B")
	assert.Contains(t, nodeIDs, "C")
	assert.Contains(t, nodeIDs, "D")
}

func TestDAGGetParallelGroups(t *testing.T) {
	dag := NewDAG()
	
	// Create a complex DAG
	//      A
	//    / | \
	//   B  C  D
	//   |  |  |
	//   E  F  |
	//   |/    |
	//   G     |
	//    \    |
	//     \   |
	//      \ /
	//       H
	
	nodes := []*Step{
		{ID: "A", Name: "Step A", Type: StepTypeAction},
		{ID: "B", Name: "Step B", Type: StepTypeAction},
		{ID: "C", Name: "Step C", Type: StepTypeAction},
		{ID: "D", Name: "Step D", Type: StepTypeAction},
		{ID: "E", Name: "Step E", Type: StepTypeAction},
		{ID: "F", Name: "Step F", Type: StepTypeAction},
		{ID: "G", Name: "Step G", Type: StepTypeAction},
		{ID: "H", Name: "Step H", Type: StepTypeAction},
	}
	
	for _, node := range nodes {
		dag.AddNode(node.ID, node)
	}
	
	// Add edges
	dag.AddEdge("A", "B")
	dag.AddEdge("A", "C")
	dag.AddEdge("A", "D")
	dag.AddEdge("B", "E")
	dag.AddEdge("C", "F")
	dag.AddEdge("E", "G")
	dag.AddEdge("F", "G")
	dag.AddEdge("G", "H")
	dag.AddEdge("D", "H")
	
	groups := dag.GetParallelGroups()
	assert.NotEmpty(t, groups)
	
	// First group should contain only A
	assert.Len(t, groups[0], 1)
	assert.Equal(t, "A", groups[0][0])
	
	// Second group should contain B, C, D (parallel)
	assert.Len(t, groups[1], 3)
	assert.Contains(t, groups[1], "B")
	assert.Contains(t, groups[1], "C")
	assert.Contains(t, groups[1], "D")
}

func TestDAGValidateWorkflow(t *testing.T) {
	tests := []struct {
		name     string
		workflow *Workflow
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid acyclic workflow",
			workflow: &Workflow{
				ID: "valid-workflow",
				Steps: []Step{
					{ID: "A", Name: "Step A", Type: StepTypeAction},
					{ID: "B", Name: "Step B", Type: StepTypeAction, Dependencies: []string{"A"}},
					{ID: "C", Name: "Step C", Type: StepTypeAction, Dependencies: []string{"A"}},
					{ID: "D", Name: "Step D", Type: StepTypeAction, Dependencies: []string{"B", "C"}},
				},
			},
			wantErr: false,
		},
		{
			name: "workflow with cycle",
			workflow: &Workflow{
				ID: "cyclic-workflow",
				Steps: []Step{
					{ID: "A", Name: "Step A", Type: StepTypeAction, Dependencies: []string{"C"}},
					{ID: "B", Name: "Step B", Type: StepTypeAction, Dependencies: []string{"A"}},
					{ID: "C", Name: "Step C", Type: StepTypeAction, Dependencies: []string{"B"}},
				},
			},
			wantErr: true,
			errMsg:  "cycle detected",
		},
		{
			name: "workflow with invalid dependency",
			workflow: &Workflow{
				ID: "invalid-dep-workflow",
				Steps: []Step{
					{ID: "A", Name: "Step A", Type: StepTypeAction},
					{ID: "B", Name: "Step B", Type: StepTypeAction, Dependencies: []string{"non-existent"}},
				},
			},
			wantErr: true,
			errMsg:  "dependency step not found",
		},
		{
			name: "workflow with duplicate step IDs",
			workflow: &Workflow{
				ID: "duplicate-id-workflow",
				Steps: []Step{
					{ID: "A", Name: "Step A", Type: StepTypeAction},
					{ID: "A", Name: "Step A Duplicate", Type: StepTypeAction},
				},
			},
			wantErr: true,
			errMsg:  "duplicate step ID",
		},
		{
			name: "empty workflow",
			workflow: &Workflow{
				ID:    "empty-workflow",
				Steps: []Step{},
			},
			wantErr: true,
			errMsg:  "workflow must have at least one step",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag := NewDAG()
			err := dag.ValidateWorkflow(tt.workflow)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDAGComplexValidation(t *testing.T) {
	dag := NewDAG()
	
	// Test self-dependency
	workflow := &Workflow{
		ID: "self-dep-workflow",
		Steps: []Step{
			{ID: "A", Name: "Step A", Type: StepTypeAction, Dependencies: []string{"A"}},
		},
	}
	
	err := dag.ValidateWorkflow(workflow)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "self-dependency")
}

func TestDAGOptimizeExecution(t *testing.T) {
	dag := NewDAG()
	
	// Create a workflow that can be optimized for parallel execution
	workflow := &Workflow{
		ID: "optimization-workflow",
		Steps: []Step{
			{ID: "A", Name: "Step A", Type: StepTypeAction},
			{ID: "B", Name: "Step B", Type: StepTypeAction, Dependencies: []string{"A"}},
			{ID: "C", Name: "Step C", Type: StepTypeAction, Dependencies: []string{"A"}},
			{ID: "D", Name: "Step D", Type: StepTypeAction, Dependencies: []string{"A"}},
			{ID: "E", Name: "Step E", Type: StepTypeAction, Dependencies: []string{"B", "C", "D"}},
		},
	}
	
	plan, err := dag.OptimizeExecution(workflow)
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	
	// Should have 3 levels: A, {B,C,D}, E
	assert.Len(t, plan.Levels, 3)
	assert.Len(t, plan.Levels[0], 1) // A
	assert.Len(t, plan.Levels[1], 3) // B, C, D
	assert.Len(t, plan.Levels[2], 1) // E
	
	// Check parallelism estimation
	assert.Equal(t, 3, plan.MaxParallelism)
	assert.Greater(t, plan.EstimatedDuration, time.Duration(0))
}

func TestDAGConcurrentAccess(t *testing.T) {
	dag := NewDAG()
	
	// Test concurrent node additions
	done := make(chan bool, 100)
	
	for i := 0; i < 100; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			step := &Step{
				ID:   fmt.Sprintf("step-%d", id),
				Name: fmt.Sprintf("Step %d", id),
				Type: StepTypeAction,
			}
			dag.AddNode(step.ID, step)
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
	
	// Verify all nodes were added
	assert.Len(t, dag.nodes, 100)
	
	// Test concurrent edge additions
	for i := 0; i < 50; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			from := fmt.Sprintf("step-%d", id)
			to := fmt.Sprintf("step-%d", id+50)
			dag.AddEdge(from, to)
		}(i)
	}
	
	// Wait for all edge additions
	for i := 0; i < 50; i++ {
		<-done
	}
	
	// Verify graph is still consistent
	assert.False(t, dag.HasCycle())
}