package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/liliang-cn/rago/pkg/tools"
)

// FileOperationTool provides file system operations for the RAG system
type FileOperationTool struct {
	allowedPaths []string // Whitelist of allowed paths for security
	maxFileSize  int64    // Maximum file size in bytes
}

// NewFileOperationTool creates a new file operation tool
func NewFileOperationTool(allowedPaths []string, maxFileSize int64) *FileOperationTool {
	if maxFileSize <= 0 {
		maxFileSize = 10 * 1024 * 1024 // Default 10MB
	}
	return &FileOperationTool{
		allowedPaths: allowedPaths,
		maxFileSize:  maxFileSize,
	}
}

// Name returns the tool name
func (t *FileOperationTool) Name() string {
	return "file_operations"
}

// Description returns the tool description
func (t *FileOperationTool) Description() string {
	return "Perform file system operations including reading, writing, and listing files within allowed directories"
}

// Parameters returns the tool parameters schema
func (t *FileOperationTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"action": {
				Type:        "string",
				Description: "The file operation to perform",
				Enum:        []string{"read", "write", "list", "exists", "stat", "delete"},
			},
			"path": {
				Type:        "string",
				Description: "The file or directory path to operate on",
			},
			"content": {
				Type:        "string",
				Description: "Content to write to file (required for 'write' action)",
			},
			"recursive": {
				Type:        "boolean",
				Description: "Whether to list directories recursively (for 'list' action)",
				Default:     false,
			},
			"pattern": {
				Type:        "string",
				Description: "File pattern to match when listing (e.g., '*.txt', '*.md')",
			},
			"max_lines": {
				Type:        "integer",
				Description: "Maximum number of lines to read (for 'read' action)",
				Minimum:     func() *float64 { v := float64(1); return &v }(),
				Maximum:     func() *float64 { v := float64(10000); return &v }(),
			},
		},
		Required: []string{"action", "path"},
	}
}

// Execute runs the file operation tool
func (t *FileOperationTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	action, ok := args["action"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "action parameter is required",
		}, nil
	}

	path, ok := args["path"].(string)
	if !ok || path == "" {
		return &tools.ToolResult{
			Success: false,
			Error:   "path parameter is required",
		}, nil
	}

	// Security check: validate path
	if !t.isPathAllowed(path) {
		return &tools.ToolResult{
			Success: false,
			Error:   "access to this path is not allowed",
		}, nil
	}

	switch action {
	case "read":
		return t.readFile(path, args)
	case "write":
		return t.writeFile(path, args)
	case "list":
		return t.listFiles(path, args)
	case "exists":
		return t.checkExists(path)
	case "stat":
		return t.getFileStat(path)
	case "delete":
		return t.deleteFile(path)
	default:
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", action),
		}, nil
	}
}

// Validate validates the tool arguments
func (t *FileOperationTool) Validate(args map[string]interface{}) error {
	action, ok := args["action"]
	if !ok {
		return fmt.Errorf("action parameter is required")
	}

	actionStr, ok := action.(string)
	if !ok {
		return fmt.Errorf("action must be a string")
	}

	validActions := []string{"read", "write", "list", "exists", "stat", "delete"}
	valid := false
	for _, v := range validActions {
		if actionStr == v {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid action: %s", actionStr)
	}

	if _, ok := args["path"]; !ok {
		return fmt.Errorf("path parameter is required")
	}

	if actionStr == "write" {
		if _, ok := args["content"]; !ok {
			return fmt.Errorf("content parameter is required for write action")
		}
	}

	return nil
}

// isPathAllowed checks if the given path is within allowed directories
func (t *FileOperationTool) isPathAllowed(path string) bool {
	if len(t.allowedPaths) == 0 {
		return false // No paths allowed if not configured
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check against allowed paths
	for _, allowedPath := range t.allowedPaths {
		absAllowed, err := filepath.Abs(allowedPath)
		if err != nil {
			continue
		}

		// Check if path is within allowed directory
		if strings.HasPrefix(absPath, absAllowed+string(filepath.Separator)) || absPath == absAllowed {
			return true
		}
	}

	return false
}

// readFile reads content from a file
func (t *FileOperationTool) readFile(path string, args map[string]interface{}) (*tools.ToolResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to access file: %v", err),
		}, nil
	}

	if info.IsDir() {
		return &tools.ToolResult{
			Success: false,
			Error:   "path is a directory, not a file",
		}, nil
	}

	if info.Size() > t.maxFileSize {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("file size (%d bytes) exceeds maximum allowed size (%d bytes)", info.Size(), t.maxFileSize),
		}, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}, nil
	}

	contentStr := string(content)

	// Apply line limit if specified
	if maxLines, ok := args["max_lines"]; ok {
		if maxLinesInt, ok := maxLines.(int); ok {
			lines := strings.Split(contentStr, "\n")
			if len(lines) > maxLinesInt {
				lines = lines[:maxLinesInt]
				contentStr = strings.Join(lines, "\n") + "\n... (truncated)"
			}
		} else if maxLinesFloat, ok := maxLines.(float64); ok {
			maxLinesInt := int(maxLinesFloat)
			lines := strings.Split(contentStr, "\n")
			if len(lines) > maxLinesInt {
				lines = lines[:maxLinesInt]
				contentStr = strings.Join(lines, "\n") + "\n... (truncated)"
			}
		}
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"content":    contentStr,
			"size":       info.Size(),
			"modified":   info.ModTime().Format(time.RFC3339),
			"path":       path,
			"mime_type":  t.detectMimeType(path),
		},
	}, nil
}

// writeFile writes content to a file
func (t *FileOperationTool) writeFile(path string, args map[string]interface{}) (*tools.ToolResult, error) {
	content, ok := args["content"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "content parameter is required for write action",
		}, nil
	}

	if int64(len(content)) > t.maxFileSize {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("content size (%d bytes) exceeds maximum allowed size (%d bytes)", len(content), t.maxFileSize),
		}, nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create directory: %v", err),
		}, nil
	}

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, nil
	}

	info, _ := os.Stat(path)
	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"path":     path,
			"size":     len(content),
			"modified": info.ModTime().Format(time.RFC3339),
			"message":  "file written successfully",
		},
	}, nil
}

// listFiles lists files in a directory
func (t *FileOperationTool) listFiles(path string, args map[string]interface{}) (*tools.ToolResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to access path: %v", err),
		}, nil
	}

	if !info.IsDir() {
		return &tools.ToolResult{
			Success: false,
			Error:   "path is not a directory",
		}, nil
	}

	recursive := false
	if rec, ok := args["recursive"].(bool); ok {
		recursive = rec
	}

	pattern := "*"
	if pat, ok := args["pattern"].(string); ok && pat != "" {
		pattern = pat
	}

	var files []map[string]interface{}

	if recursive {
			if err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip files with errors
			}

			// Skip if path is not allowed (for security)
			if !t.isPathAllowed(filePath) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			matched, _ := filepath.Match(pattern, d.Name())
			if matched {
				info, err := d.Info()
				if err != nil {
					return nil
				}

				files = append(files, map[string]interface{}{
					"name":     d.Name(),
					"path":     filePath,
					"type":     t.getFileType(d),
					"size":     info.Size(),
					"modified": info.ModTime().Format(time.RFC3339),
				})
			}
			return nil
		}); err != nil {
			return &tools.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to walk directory: %v", err),
			}, nil
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return &tools.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to read directory: %v", err),
			}, nil
		}

		for _, entry := range entries {
			matched, _ := filepath.Match(pattern, entry.Name())
			if matched {
				info, err := entry.Info()
				if err != nil {
					continue
				}

				filePath := filepath.Join(path, entry.Name())
				files = append(files, map[string]interface{}{
					"name":     entry.Name(),
					"path":     filePath,
					"type":     t.getFileType(entry),
					"size":     info.Size(),
					"modified": info.ModTime().Format(time.RFC3339),
				})
			}
		}
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"path":      path,
			"files":     files,
			"count":     len(files),
			"recursive": recursive,
			"pattern":   pattern,
		},
	}, nil
}

// checkExists checks if a file or directory exists
func (t *FileOperationTool) checkExists(path string) (*tools.ToolResult, error) {
	info, err := os.Stat(path)
	exists := !os.IsNotExist(err)

	result := map[string]interface{}{
		"path":   path,
		"exists": exists,
	}

	if exists && err == nil {
		result["type"] = t.getFileTypeFromInfo(info)
		result["size"] = info.Size()
		result["modified"] = info.ModTime().Format(time.RFC3339)
	}

	return &tools.ToolResult{
		Success: true,
		Data:    result,
	}, nil
}

// getFileStat gets detailed file statistics
func (t *FileOperationTool) getFileStat(path string) (*tools.ToolResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to get file info: %v", err),
		}, nil
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"path":        path,
			"name":        info.Name(),
			"size":        info.Size(),
			"type":        t.getFileTypeFromInfo(info),
			"mode":        info.Mode().String(),
			"permissions": fmt.Sprintf("%o", info.Mode().Perm()),
			"modified":    info.ModTime().Format(time.RFC3339),
			"is_dir":      info.IsDir(),
			"mime_type":   t.detectMimeType(path),
		},
	}, nil
}

// deleteFile deletes a file
func (t *FileOperationTool) deleteFile(path string) (*tools.ToolResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &tools.ToolResult{
				Success: false,
				Error:   "file does not exist",
			}, nil
		}
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to access file: %v", err),
		}, nil
	}

	err = os.Remove(path)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to delete file: %v", err),
		}, nil
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"path":    path,
			"message": fmt.Sprintf("%s deleted successfully", t.getFileTypeFromInfo(info)),
		},
	}, nil
}

// getFileType returns the type of file system entry
func (t *FileOperationTool) getFileType(entry fs.DirEntry) string {
	if entry.IsDir() {
		return "directory"
	}
	return "file"
}

// getFileTypeFromInfo returns the type from FileInfo
func (t *FileOperationTool) getFileTypeFromInfo(info os.FileInfo) string {
	if info.IsDir() {
		return "directory"
	}
	return "file"
}

// detectMimeType attempts to detect the MIME type of a file
func (t *FileOperationTool) detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	mimeMap := map[string]string{
		".txt":  "text/plain",
		".md":   "text/markdown",
		".json": "application/json",
		".xml":  "application/xml",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".pdf":  "application/pdf",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".csv":  "text/csv",
		".go":   "text/x-go",
		".py":   "text/x-python",
		".sql":  "text/x-sql",
	}

	if mime, exists := mimeMap[ext]; exists {
		return mime
	}
	return "application/octet-stream"
}