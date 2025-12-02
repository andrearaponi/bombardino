package variables

import (
	"fmt"
)

// TestDependency represents a test with its dependencies
type TestDependency struct {
	Name      string
	DependsOn []string
}

// ExecutionPlan represents the order in which tests should be executed
type ExecutionPlan struct {
	Phases [][]string // Each phase contains tests that can run in parallel
}

// BuildDAG constructs an execution plan from test dependencies using topological sort
func BuildDAG(tests []TestDependency) (*ExecutionPlan, error) {
	if len(tests) == 0 {
		return &ExecutionPlan{Phases: [][]string{}}, nil
	}

	// Build adjacency list and in-degree count
	testNames := make(map[string]bool)
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // who depends on this test

	// Initialize all tests
	for _, test := range tests {
		testNames[test.Name] = true
		if _, ok := inDegree[test.Name]; !ok {
			inDegree[test.Name] = 0
		}
	}

	// Build dependency graph
	for _, test := range tests {
		for _, dep := range test.DependsOn {
			// Check if dependency exists
			if !testNames[dep] {
				return nil, fmt.Errorf("unknown dependency: test '%s' depends on '%s' which doesn't exist", test.Name, dep)
			}
			inDegree[test.Name]++
			dependents[dep] = append(dependents[dep], test.Name)
		}
	}

	// Kahn's algorithm for topological sort with level tracking
	var phases [][]string
	processed := 0
	totalTests := len(tests)

	for processed < totalTests {
		// Find all tests with no remaining dependencies (in-degree = 0)
		var currentPhase []string
		for name := range testNames {
			if inDegree[name] == 0 {
				currentPhase = append(currentPhase, name)
			}
		}

		// If no tests can be processed, we have a cycle
		if len(currentPhase) == 0 {
			return nil, fmt.Errorf("cyclic dependency detected in tests")
		}

		// Process current phase
		for _, name := range currentPhase {
			delete(testNames, name) // Remove from remaining tests
			processed++

			// Decrease in-degree of dependents
			for _, dependent := range dependents[name] {
				inDegree[dependent]--
			}
		}

		// Remove processed tests from inDegree
		for _, name := range currentPhase {
			delete(inDegree, name)
		}

		phases = append(phases, currentPhase)
	}

	return &ExecutionPlan{Phases: phases}, nil
}

// GetPhaseForTest returns the phase index for a given test name
func (ep *ExecutionPlan) GetPhaseForTest(testName string) int {
	for i, phase := range ep.Phases {
		for _, name := range phase {
			if name == testName {
				return i
			}
		}
	}
	return -1
}

// TotalPhases returns the number of phases in the execution plan
func (ep *ExecutionPlan) TotalPhases() int {
	return len(ep.Phases)
}

// AllTests returns all test names in execution order (flattened)
func (ep *ExecutionPlan) AllTests() []string {
	var result []string
	for _, phase := range ep.Phases {
		result = append(result, phase...)
	}
	return result
}
