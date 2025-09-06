// Package workflow provides state storage for workflow persistence.
package workflow

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// StateStorage provides persistent storage for workflow state.
type StateStorage interface {
	SaveWorkflow(def *Definition) error
	LoadWorkflow(id string) (*Definition, error)
	DeleteWorkflow(id string) error
	
	SaveExecutionState(ctx *ExecutionContext) error
	LoadExecutionState(id string) (*ExecutionContext, error)
	DeleteExecutionState(id string) error
	
	ListWorkflows() ([]*Definition, error)
	ListExecutions() ([]*ExecutionContext, error)
	
	Close() error
}

// FileStateStorage implements StateStorage using the file system.
type FileStateStorage struct {
	mu           sync.RWMutex
	basePath     string
	workflowPath string
	executionPath string
}

// NewStateStorage creates a new state storage instance.
func NewStateStorage(basePath string) (StateStorage, error) {
	if basePath == "" {
		basePath = "/tmp/rago/workflow_state"
	}
	
	storage := &FileStateStorage{
		basePath:      basePath,
		workflowPath:  filepath.Join(basePath, "workflows"),
		executionPath: filepath.Join(basePath, "executions"),
	}
	
	// Create directories if they don't exist
	if err := os.MkdirAll(storage.workflowPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workflow directory: %w", err)
	}
	
	if err := os.MkdirAll(storage.executionPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create execution directory: %w", err)
	}
	
	return storage, nil
}

// SaveWorkflow saves a workflow definition to storage.
func (fs *FileStateStorage) SaveWorkflow(def *Definition) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	data, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}
	
	filename := filepath.Join(fs.workflowPath, fmt.Sprintf("%s.json", def.ID))
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}
	
	return nil
}

// LoadWorkflow loads a workflow definition from storage.
func (fs *FileStateStorage) LoadWorkflow(id string) (*Definition, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	filename := filepath.Join(fs.workflowPath, fmt.Sprintf("%s.json", id))
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("workflow %s not found", id)
		}
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}
	
	var def Definition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow: %w", err)
	}
	
	return &def, nil
}

// DeleteWorkflow deletes a workflow definition from storage.
func (fs *FileStateStorage) DeleteWorkflow(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	filename := filepath.Join(fs.workflowPath, fmt.Sprintf("%s.json", id))
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("workflow %s not found", id)
		}
		return fmt.Errorf("failed to delete workflow file: %w", err)
	}
	
	return nil
}

// SaveExecutionState saves execution state to storage.
func (fs *FileStateStorage) SaveExecutionState(ctx *ExecutionContext) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal execution state: %w", err)
	}
	
	filename := filepath.Join(fs.executionPath, fmt.Sprintf("%s.json", ctx.ID))
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write execution file: %w", err)
	}
	
	return nil
}

// LoadExecutionState loads execution state from storage.
func (fs *FileStateStorage) LoadExecutionState(id string) (*ExecutionContext, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	filename := filepath.Join(fs.executionPath, fmt.Sprintf("%s.json", id))
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("execution %s not found", id)
		}
		return nil, fmt.Errorf("failed to read execution file: %w", err)
	}
	
	var ctx ExecutionContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution state: %w", err)
	}
	
	return &ctx, nil
}

// DeleteExecutionState deletes execution state from storage.
func (fs *FileStateStorage) DeleteExecutionState(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	filename := filepath.Join(fs.executionPath, fmt.Sprintf("%s.json", id))
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("execution %s not found", id)
		}
		return fmt.Errorf("failed to delete execution file: %w", err)
	}
	
	return nil
}

// ListWorkflows lists all workflow definitions.
func (fs *FileStateStorage) ListWorkflows() ([]*Definition, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	files, err := ioutil.ReadDir(fs.workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow directory: %w", err)
	}
	
	workflows := make([]*Definition, 0)
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		
		data, err := ioutil.ReadFile(filepath.Join(fs.workflowPath, file.Name()))
		if err != nil {
			continue
		}
		
		var def Definition
		if err := json.Unmarshal(data, &def); err != nil {
			continue
		}
		
		workflows = append(workflows, &def)
	}
	
	return workflows, nil
}

// ListExecutions lists all execution states.
func (fs *FileStateStorage) ListExecutions() ([]*ExecutionContext, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	files, err := ioutil.ReadDir(fs.executionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read execution directory: %w", err)
	}
	
	executions := make([]*ExecutionContext, 0)
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		
		data, err := ioutil.ReadFile(filepath.Join(fs.executionPath, file.Name()))
		if err != nil {
			continue
		}
		
		var ctx ExecutionContext
		if err := json.Unmarshal(data, &ctx); err != nil {
			continue
		}
		
		executions = append(executions, &ctx)
	}
	
	return executions, nil
}

// Close closes the storage.
func (fs *FileStateStorage) Close() error {
	// Nothing to close for file storage
	return nil
}