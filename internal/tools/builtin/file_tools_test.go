package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileOperationTool_Name(t *testing.T) {
	tool := NewFileOperationTool([]string{"/tmp"}, 1024)
	assert.Equal(t, "file_operations", tool.Name())
}

func TestFileOperationTool_Description(t *testing.T) {
	tool := NewFileOperationTool([]string{"/tmp"}, 1024)
	assert.NotEmpty(t, tool.Description())
}

func TestFileOperationTool_Parameters(t *testing.T) {
	tool := NewFileOperationTool([]string{"/tmp"}, 1024)
	params := tool.Parameters()

	assert.Equal(t, "object", params.Type)
	assert.Contains(t, params.Required, "action")
	assert.Contains(t, params.Required, "path")
	assert.Contains(t, params.Properties, "action")
	assert.Contains(t, params.Properties, "path")
}

func TestFileOperationTool_Validate(t *testing.T) {
	tool := NewFileOperationTool([]string{"/tmp"}, 1024)

	// Valid case
	err := tool.Validate(map[string]interface{}{
		"action": "read",
		"path":   "/tmp/test.txt",
	})
	assert.NoError(t, err)

	// Missing action
	err = tool.Validate(map[string]interface{}{
		"path": "/tmp/test.txt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action parameter is required")

	// Missing path
	err = tool.Validate(map[string]interface{}{
		"action": "read",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path parameter is required")

	// Invalid action
	err = tool.Validate(map[string]interface{}{
		"action": "invalid",
		"path":   "/tmp/test.txt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action")

	// Write action without content
	err = tool.Validate(map[string]interface{}{
		"action": "write",
		"path":   "/tmp/test.txt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content parameter is required")

	// Write action with content
	err = tool.Validate(map[string]interface{}{
		"action":  "write",
		"path":    "/tmp/test.txt",
		"content": "test content",
	})
	assert.NoError(t, err)
}

func TestFileOperationTool_IsPathAllowed(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewFileOperationTool([]string{tmpDir}, 1024)

	// Test allowed path
	allowedPath := filepath.Join(tmpDir, "test.txt")
	assert.True(t, tool.isPathAllowed(allowedPath))

	// Test subdirectory
	subdir := filepath.Join(tmpDir, "subdir", "test.txt")
	assert.True(t, tool.isPathAllowed(subdir))

	// Test disallowed path
	disallowedPath := "/etc/passwd"
	assert.False(t, tool.isPathAllowed(disallowedPath))

	// Test empty allowed paths
	toolEmpty := NewFileOperationTool([]string{}, 1024)
	assert.False(t, toolEmpty.isPathAllowed(allowedPath))
}

func TestFileOperationTool_WriteAndRead(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewFileOperationTool([]string{tmpDir}, 1024)
	ctx := context.Background()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file."

	// Test write
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":  "write",
		"path":    testFile,
		"content": testContent,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, testFile, data["path"])
	assert.Equal(t, len(testContent), data["size"])

	// Test read
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action": "read",
		"path":   testFile,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok = result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, testContent, data["content"])
	assert.Equal(t, testFile, data["path"])
	assert.Contains(t, data, "size")
	assert.Contains(t, data, "modified")
}

func TestFileOperationTool_ReadWithLineLimit(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewFileOperationTool([]string{tmpDir}, 1024)
	ctx := context.Background()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"

	// Write test file
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":  "write",
		"path":    testFile,
		"content": testContent,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Read with line limit
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action":    "read",
		"path":      testFile,
		"max_lines": 3,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	content := data["content"].(string)
	assert.Contains(t, content, "Line 1")
	assert.Contains(t, content, "Line 2")
	assert.Contains(t, content, "Line 3")
	assert.Contains(t, content, "truncated")
	assert.NotContains(t, content, "Line 4")
}

func TestFileOperationTool_List(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewFileOperationTool([]string{tmpDir}, 1024)
	ctx := context.Background()

	// Create test files
	testFiles := []string{"test1.txt", "test2.md", "test3.txt"}
	for _, filename := range testFiles {
		filepath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filepath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// Create subdirectory with file
	subdir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subdir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subdir, "subfile.txt"), []byte("sub content"), 0644)
	require.NoError(t, err)

	// Test list directory
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "list",
		"path":   tmpDir,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	files, ok := data["files"].([]map[string]interface{})
	require.True(t, ok)
	assert.Len(t, files, 4) // 3 files + 1 subdirectory

	// Test list with pattern
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action":  "list",
		"path":    tmpDir,
		"pattern": "*.txt",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok = result.Data.(map[string]interface{})
	require.True(t, ok)
	files, ok = data["files"].([]map[string]interface{})
	require.True(t, ok)
	assert.Len(t, files, 2) // Only .txt files

	// Test recursive list
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action":    "list",
		"path":      tmpDir,
		"recursive": true,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok = result.Data.(map[string]interface{})
	require.True(t, ok)
	files, ok = data["files"].([]map[string]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(files), 4) // Should include subdirectory files
}

func TestFileOperationTool_Exists(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewFileOperationTool([]string{tmpDir}, 1024)
	ctx := context.Background()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Test non-existent file
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "exists",
		"path":   testFile,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.False(t, data["exists"].(bool))

	// Create file
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Test existing file
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action": "exists",
		"path":   testFile,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok = result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, data["exists"].(bool))
	assert.Equal(t, "file", data["type"])
}

func TestFileOperationTool_Stat(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewFileOperationTool([]string{tmpDir}, 1024)
	ctx := context.Background()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	// Create test file
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Test stat
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "stat",
		"path":   testFile,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test.txt", data["name"])
	assert.Equal(t, int64(len(testContent)), data["size"])
	assert.Equal(t, "file", data["type"])
	assert.False(t, data["is_dir"].(bool))
	assert.Contains(t, data, "modified")
	assert.Contains(t, data, "mode")
	assert.Contains(t, data, "permissions")
}

func TestFileOperationTool_Delete(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewFileOperationTool([]string{tmpDir}, 1024)
	ctx := context.Background()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(testFile)
	assert.NoError(t, err)

	// Test delete
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "delete",
		"path":   testFile,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify file no longer exists
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))
}

func TestFileOperationTool_SecurityChecks(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewFileOperationTool([]string{tmpDir}, 1024)
	ctx := context.Background()

	// Test access to disallowed path
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "read",
		"path":   "/etc/passwd",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not allowed")
}

func TestFileOperationTool_FileSizeLimit(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewFileOperationTool([]string{tmpDir}, 10) // 10 bytes limit
	ctx := context.Background()
	testFile := filepath.Join(tmpDir, "test.txt")
	largeContent := "This content is definitely longer than 10 bytes"

	// Test write with content exceeding limit
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":  "write",
		"path":    testFile,
		"content": largeContent,
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "exceeds maximum allowed size")
}

func TestFileOperationTool_DetectMimeType(t *testing.T) {
	tool := NewFileOperationTool([]string{"/tmp"}, 1024)

	tests := []struct {
		filename string
		expected string
	}{
		{"test.txt", "text/plain"},
		{"test.md", "text/markdown"},
		{"test.json", "application/json"},
		{"test.pdf", "application/pdf"},
		{"test.png", "image/png"},
		{"test.jpg", "image/jpeg"},
		{"test.unknown", "application/octet-stream"},
	}

	for _, test := range tests {
		mime := tool.detectMimeType(test.filename)
		assert.Equal(t, test.expected, mime, "Failed for %s", test.filename)
	}
}

func TestFileOperationTool_ErrorCases(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_tool_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewFileOperationTool([]string{tmpDir}, 1024)
	ctx := context.Background()

	// Test missing action parameter
	result, err := tool.Execute(ctx, map[string]interface{}{
		"path": "/tmp/test.txt",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "action parameter is required")

	// Test missing path parameter
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action": "read",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "path parameter is required")

	// Test reading non-existent file
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action": "read",
		"path":   filepath.Join(tmpDir, "nonexistent.txt"),
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "failed to access file")

	// Test reading directory as file
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action": "read",
		"path":   tmpDir,
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "path is a directory")

	// Test listing non-existent directory
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action": "list",
		"path":   filepath.Join(tmpDir, "nonexistent"),
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "failed to access path")

	// Test deleting non-existent file
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action": "delete",
		"path":   filepath.Join(tmpDir, "nonexistent.txt"),
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "does not exist")
}