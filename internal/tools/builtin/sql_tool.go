package builtin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/liliang-cn/rago/internal/tools"
	_ "modernc.org/sqlite" // SQLite driver
)

// SQLQueryTool provides SQL query capabilities for the RAG system
type SQLQueryTool struct {
	allowedDBs   map[string]string // Map of database names to connection strings
	maxRows      int               // Maximum number of rows to return
	queryTimeout time.Duration     // Query execution timeout
}

// NewSQLQueryTool creates a new SQL query tool
func NewSQLQueryTool(allowedDBs map[string]string, maxRows int, queryTimeout time.Duration) *SQLQueryTool {
	if maxRows <= 0 {
		maxRows = 1000 // Default max rows
	}
	if queryTimeout <= 0 {
		queryTimeout = 30 * time.Second // Default timeout
	}
	return &SQLQueryTool{
		allowedDBs:   allowedDBs,
		maxRows:      maxRows,
		queryTimeout: queryTimeout,
	}
}

// Name returns the tool name
func (t *SQLQueryTool) Name() string {
	return "sql_query"
}

// Description returns the tool description
func (t *SQLQueryTool) Description() string {
	return "Execute SQL queries against configured databases with security restrictions and result limits"
}

// Parameters returns the tool parameters schema
func (t *SQLQueryTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"action": {
				Type:        "string",
				Description: "The SQL operation to perform",
				Enum:        []string{"query", "schema", "tables", "describe"},
			},
			"database": {
				Type:        "string",
				Description: "The database name to query against",
			},
			"sql": {
				Type:        "string",
				Description: "The SQL query to execute (required for 'query' action)",
			},
			"table": {
				Type:        "string",
				Description: "Table name to describe (required for 'describe' action)",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of rows to return (default: 100)",
				Minimum:     func() *float64 { v := float64(1); return &v }(),
				Maximum:     func() *float64 { v := float64(1000); return &v }(),
				Default:     100,
			},
		},
		Required: []string{"action", "database"},
	}
}

// Execute runs the SQL query tool
func (t *SQLQueryTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	action, ok := args["action"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "action parameter is required",
		}, nil
	}

	dbName, ok := args["database"].(string)
	if !ok || dbName == "" {
		return &tools.ToolResult{
			Success: false,
			Error:   "database parameter is required",
		}, nil
	}

	// Security check: validate database access
	connectionString, exists := t.allowedDBs[dbName]
	if !exists {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("access to database '%s' is not allowed", dbName),
		}, nil
	}

	// Open database connection
	db, err := sql.Open("sqlite", connectionString)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to connect to database: %v", err),
		}, nil
	}
	defer db.Close()

	// Set connection timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, t.queryTimeout)
	defer cancel()

	switch action {
	case "query":
		return t.executeQuery(ctxWithTimeout, db, args)
	case "schema":
		return t.getSchema(ctxWithTimeout, db)
	case "tables":
		return t.getTables(ctxWithTimeout, db)
	case "describe":
		return t.describeTable(ctxWithTimeout, db, args)
	default:
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", action),
		}, nil
	}
}

// Validate validates the tool arguments
func (t *SQLQueryTool) Validate(args map[string]interface{}) error {
	action, ok := args["action"]
	if !ok {
		return fmt.Errorf("action parameter is required")
	}

	actionStr, ok := action.(string)
	if !ok {
		return fmt.Errorf("action must be a string")
	}

	validActions := []string{"query", "schema", "tables", "describe"}
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

	if _, ok := args["database"]; !ok {
		return fmt.Errorf("database parameter is required")
	}

	if actionStr == "query" {
		if _, ok := args["sql"]; !ok {
			return fmt.Errorf("sql parameter is required for query action")
		}
		
		// Basic SQL injection protection
		if sqlStr, ok := args["sql"].(string); ok {
			if err := t.validateSQL(sqlStr); err != nil {
				return fmt.Errorf("invalid SQL: %v", err)
			}
		}
	}

	if actionStr == "describe" {
		if _, ok := args["table"]; !ok {
			return fmt.Errorf("table parameter is required for describe action")
		}
	}

	return nil
}

// validateSQL provides basic SQL injection protection
func (t *SQLQueryTool) validateSQL(query string) error {
	query = strings.ToLower(strings.TrimSpace(query))
	
	// Block dangerous operations
	dangerousKeywords := []string{
		"drop", "delete", "update", "insert", "alter", "create", "truncate",
		"exec", "execute", "sp_", "xp_", "into outfile", "load_file",
		"information_schema", "mysql", "pg_", "sqlite_master",
	}
	
	for _, keyword := range dangerousKeywords {
		if strings.Contains(query, keyword) {
			return fmt.Errorf("potentially dangerous SQL keyword detected: %s", keyword)
		}
	}
	
	// Only allow SELECT statements
	if !strings.HasPrefix(query, "select") {
		return fmt.Errorf("only SELECT statements are allowed")
	}
	
	return nil
}

// executeQuery executes a SQL query and returns results
func (t *SQLQueryTool) executeQuery(ctx context.Context, db *sql.DB, args map[string]interface{}) (*tools.ToolResult, error) {
	sqlQuery, ok := args["sql"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "sql parameter is required for query action",
		}, nil
	}

	// Get limit
	limit := 100
	if l, ok := args["limit"]; ok {
		if limitInt, ok := l.(int); ok {
			limit = limitInt
		} else if limitFloat, ok := l.(float64); ok {
			limit = int(limitFloat)
		}
	}
	
	if limit > t.maxRows {
		limit = t.maxRows
	}

	// Add LIMIT clause if not present
	if !strings.Contains(strings.ToLower(sqlQuery), "limit") {
		sqlQuery = fmt.Sprintf("%s LIMIT %d", sqlQuery, limit)
	}

	start := time.Now()
	rows, err := db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("query execution failed: %v", err),
		}, nil
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to get columns: %v", err),
		}, nil
	}

	// Prepare result structure
	var results []map[string]interface{}
	
	// Create value holders
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	rowCount := 0
	for rows.Next() && rowCount < limit {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return &tools.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to scan row: %v", err),
			}, nil
		}

		// Convert row to map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			
			// Convert byte arrays to strings
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			
			row[col] = val
		}
		
		results = append(results, row)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("row iteration error: %v", err),
		}, nil
	}

	elapsed := time.Since(start)

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"columns":     columns,
			"rows":        results,
			"row_count":   len(results),
			"query":       sqlQuery,
			"elapsed":     elapsed.String(),
			"limited":     len(results) >= limit,
		},
	}, nil
}

// getSchema returns database schema information
func (t *SQLQueryTool) getSchema(ctx context.Context, db *sql.DB) (*tools.ToolResult, error) {
	query := `
		SELECT 
			name as table_name,
			type,
			sql as create_statement
		FROM sqlite_master 
		WHERE type IN ('table', 'view') 
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to get schema: %v", err),
		}, nil
	}
	defer rows.Close()

	var tables []map[string]interface{}
	for rows.Next() {
		var name, objType, createSQL string
		err := rows.Scan(&name, &objType, &createSQL)
		if err != nil {
			continue
		}

		tables = append(tables, map[string]interface{}{
			"name":             name,
			"type":             objType,
			"create_statement": createSQL,
		})
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"tables": tables,
			"count":  len(tables),
		},
	}, nil
}

// getTables returns list of tables in the database
func (t *SQLQueryTool) getTables(ctx context.Context, db *sql.DB) (*tools.ToolResult, error) {
	query := `
		SELECT name 
		FROM sqlite_master 
		WHERE type = 'table' 
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to get tables: %v", err),
		}, nil
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			continue
		}
		tables = append(tables, name)
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"tables": tables,
			"count":  len(tables),
		},
	}, nil
}

// describeTable returns column information for a specific table
func (t *SQLQueryTool) describeTable(ctx context.Context, db *sql.DB, args map[string]interface{}) (*tools.ToolResult, error) {
	tableName, ok := args["table"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "table parameter is required for describe action",
		}, nil
	}

	// Validate table name to prevent injection
	if err := t.validateTableName(tableName); err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid table name: %v", err),
		}, nil
	}

	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to describe table: %v", err),
		}, nil
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue interface{}

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}

		column := map[string]interface{}{
			"position":     cid,
			"name":         name,
			"type":         dataType,
			"not_null":     notNull == 1,
			"primary_key":  pk == 1,
		}

		if defaultValue != nil {
			column["default_value"] = defaultValue
		}

		columns = append(columns, column)
	}

	if len(columns) == 0 {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("table '%s' not found or has no columns", tableName),
		}, nil
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"table":   tableName,
			"columns": columns,
			"count":   len(columns),
		},
	}, nil
}

// validateTableName ensures table name is safe
func (t *SQLQueryTool) validateTableName(tableName string) error {
	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// Allow only alphanumeric characters and underscores
	for _, r := range tableName {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return fmt.Errorf("table name contains invalid characters")
		}
	}

	// Prevent SQL injection through table names
	lowerName := strings.ToLower(tableName)
	if strings.Contains(lowerName, "drop") || strings.Contains(lowerName, "delete") ||
		strings.Contains(lowerName, "update") || strings.Contains(lowerName, "insert") {
		return fmt.Errorf("table name contains dangerous keywords")
	}

	return nil
}

// GetSupportedDatabases returns list of configured databases
func (t *SQLQueryTool) GetSupportedDatabases() []string {
	var dbs []string
	for dbName := range t.allowedDBs {
		dbs = append(dbs, dbName)
	}
	return dbs
}