package builtin

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (string, func()) {
	// Create temporary database file
	tmpDir, err := os.MkdirTemp("", "sql_tool_test")
	require.NoError(t, err)
	
	dbPath := filepath.Join(tmpDir, "test.db")
	
	// Create and setup test database
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	
	// Create test tables
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT,
			age INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)
	
	_, err = db.Exec(`
		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			price REAL,
			category TEXT
		)
	`)
	require.NoError(t, err)
	
	// Insert test data
	_, err = db.Exec(`
		INSERT INTO users (name, email, age) VALUES 
		('John Doe', 'john@example.com', 30),
		('Jane Smith', 'jane@example.com', 25),
		('Bob Johnson', 'bob@example.com', 35)
	`)
	require.NoError(t, err)
	
	_, err = db.Exec(`
		INSERT INTO products (name, price, category) VALUES 
		('Laptop', 999.99, 'Electronics'),
		('Book', 19.99, 'Education'),
		('Coffee Mug', 9.99, 'Kitchen')
	`)
	require.NoError(t, err)
	
	db.Close()
	
	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}
	
	return dbPath, cleanup
}

func TestSQLQueryTool_Name(t *testing.T) {
	tool := NewSQLQueryTool(map[string]string{"test": ":memory:"}, 100, 30*time.Second)
	assert.Equal(t, "sql_query", tool.Name())
}

func TestSQLQueryTool_Description(t *testing.T) {
	tool := NewSQLQueryTool(map[string]string{"test": ":memory:"}, 100, 30*time.Second)
	assert.NotEmpty(t, tool.Description())
}

func TestSQLQueryTool_Parameters(t *testing.T) {
	tool := NewSQLQueryTool(map[string]string{"test": ":memory:"}, 100, 30*time.Second)
	params := tool.Parameters()

	assert.Equal(t, "object", params.Type)
	assert.Contains(t, params.Required, "action")
	assert.Contains(t, params.Required, "database")
	assert.Contains(t, params.Properties, "action")
	assert.Contains(t, params.Properties, "database")
	assert.Contains(t, params.Properties, "sql")
	assert.Contains(t, params.Properties, "table")
}

func TestSQLQueryTool_Validate(t *testing.T) {
	tool := NewSQLQueryTool(map[string]string{"test": ":memory:"}, 100, 30*time.Second)

	// Valid query
	err := tool.Validate(map[string]interface{}{
		"action":   "query",
		"database": "test",
		"sql":      "SELECT * FROM users",
	})
	assert.NoError(t, err)

	// Missing action
	err = tool.Validate(map[string]interface{}{
		"database": "test",
		"sql":      "SELECT * FROM users",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action parameter is required")

	// Missing database
	err = tool.Validate(map[string]interface{}{
		"action": "query",
		"sql":    "SELECT * FROM users",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database parameter is required")

	// Query action without SQL
	err = tool.Validate(map[string]interface{}{
		"action":   "query",
		"database": "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sql parameter is required")

	// Describe action without table
	err = tool.Validate(map[string]interface{}{
		"action":   "describe",
		"database": "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "table parameter is required")

	// Invalid action
	err = tool.Validate(map[string]interface{}{
		"action":   "invalid",
		"database": "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action")
}

func TestSQLQueryTool_ValidateSQL(t *testing.T) {
	tool := NewSQLQueryTool(map[string]string{"test": ":memory:"}, 100, 30*time.Second)

	// Valid SELECT
	err := tool.validateSQL("SELECT * FROM users")
	assert.NoError(t, err)

	// Valid SELECT with WHERE
	err = tool.validateSQL("SELECT name, email FROM users WHERE age > 25")
	assert.NoError(t, err)

	// Dangerous DROP statement
	err = tool.validateSQL("DROP TABLE users")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dangerous SQL keyword")

	// DELETE statement
	err = tool.validateSQL("DELETE FROM users")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dangerous SQL keyword")

	// UPDATE statement
	err = tool.validateSQL("UPDATE users SET name = 'test'")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dangerous SQL keyword")

	// Non-SELECT statement
	err = tool.validateSQL("INSERT INTO users VALUES (1, 'test')")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dangerous SQL keyword")
}

func TestSQLQueryTool_ValidateTableName(t *testing.T) {
	tool := NewSQLQueryTool(map[string]string{"test": ":memory:"}, 100, 30*time.Second)

	// Valid table names
	assert.NoError(t, tool.validateTableName("users"))
	assert.NoError(t, tool.validateTableName("user_profile"))
	assert.NoError(t, tool.validateTableName("Table123"))

	// Invalid table names
	assert.Error(t, tool.validateTableName(""))
	assert.Error(t, tool.validateTableName("users; DROP TABLE"))
	assert.Error(t, tool.validateTableName("user-profile"))
	assert.Error(t, tool.validateTableName("user.profile"))
}

func TestSQLQueryTool_ExecuteQuery(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	tool := NewSQLQueryTool(map[string]string{"test": dbPath}, 100, 30*time.Second)
	ctx := context.Background()

	// Test basic query
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":   "query",
		"database": "test",
		"sql":      "SELECT name, email FROM users ORDER BY name",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	columns := data["columns"].([]string)
	assert.Contains(t, columns, "name")
	assert.Contains(t, columns, "email")

	rows := data["rows"].([]map[string]interface{})
	assert.Len(t, rows, 3)
	assert.Equal(t, "Bob Johnson", rows[0]["name"])
	assert.Equal(t, "Jane Smith", rows[1]["name"])
	assert.Equal(t, "John Doe", rows[2]["name"])
}

func TestSQLQueryTool_ExecuteQueryWithLimit(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	tool := NewSQLQueryTool(map[string]string{"test": dbPath}, 100, 30*time.Second)
	ctx := context.Background()

	// Test query with limit
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":   "query",
		"database": "test",
		"sql":      "SELECT * FROM users",
		"limit":    2,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	rows := data["rows"].([]map[string]interface{})
	assert.LessOrEqual(t, len(rows), 2)
}

func TestSQLQueryTool_GetTables(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	tool := NewSQLQueryTool(map[string]string{"test": dbPath}, 100, 30*time.Second)
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":   "tables",
		"database": "test",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	tables := data["tables"].([]string)
	assert.Contains(t, tables, "users")
	assert.Contains(t, tables, "products")
	assert.Equal(t, 2, data["count"])
}

func TestSQLQueryTool_GetSchema(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	tool := NewSQLQueryTool(map[string]string{"test": dbPath}, 100, 30*time.Second)
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":   "schema",
		"database": "test",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	tables := data["tables"].([]map[string]interface{})
	assert.Len(t, tables, 2)

	// Check that we have both tables
	tableNames := make([]string, len(tables))
	for i, table := range tables {
		tableNames[i] = table["name"].(string)
	}
	assert.Contains(t, tableNames, "users")
	assert.Contains(t, tableNames, "products")
}

func TestSQLQueryTool_DescribeTable(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	tool := NewSQLQueryTool(map[string]string{"test": dbPath}, 100, 30*time.Second)
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":   "describe",
		"database": "test",
		"table":    "users",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "users", data["table"])
	columns := data["columns"].([]map[string]interface{})
	assert.GreaterOrEqual(t, len(columns), 4) // id, name, email, age, created_at

	// Check for expected columns
	columnNames := make([]string, len(columns))
	for i, col := range columns {
		columnNames[i] = col["name"].(string)
	}
	assert.Contains(t, columnNames, "id")
	assert.Contains(t, columnNames, "name")
	assert.Contains(t, columnNames, "email")
	assert.Contains(t, columnNames, "age")
}

func TestSQLQueryTool_SecurityChecks(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	tool := NewSQLQueryTool(map[string]string{"test": dbPath}, 100, 30*time.Second)
	ctx := context.Background()

	// Test access to non-allowed database
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":   "query",
		"database": "unauthorized",
		"sql":      "SELECT * FROM users",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not allowed")

	// Test dangerous SQL injection attempt
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action":   "query",
		"database": "test",
		"sql":      "SELECT * FROM users; DROP TABLE users;",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	// Note: The validation happens before execution, so we expect validation error
}

func TestSQLQueryTool_ErrorCases(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	tool := NewSQLQueryTool(map[string]string{"test": dbPath}, 100, 30*time.Second)
	ctx := context.Background()

	// Test missing action
	result, err := tool.Execute(ctx, map[string]interface{}{
		"database": "test",
		"sql":      "SELECT * FROM users",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "action parameter is required")

	// Test missing database
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action": "query",
		"sql":    "SELECT * FROM users",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "database parameter is required")

	// Test invalid SQL syntax
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action":   "query",
		"database": "test",
		"sql":      "SELECT * FROM nonexistent_table",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "query execution failed")

	// Test describe non-existent table
	result, err = tool.Execute(ctx, map[string]interface{}{
		"action":   "describe",
		"database": "test",
		"table":    "nonexistent_table",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not found")
}

func TestSQLQueryTool_GetSupportedDatabases(t *testing.T) {
	allowedDBs := map[string]string{
		"main": "/path/to/main.db",
		"test": "/path/to/test.db",
		"prod": "/path/to/prod.db",
	}
	
	tool := NewSQLQueryTool(allowedDBs, 100, 30*time.Second)
	dbs := tool.GetSupportedDatabases()
	
	assert.Len(t, dbs, 3)
	assert.Contains(t, dbs, "main")
	assert.Contains(t, dbs, "test")
	assert.Contains(t, dbs, "prod")
}

func TestSQLQueryTool_DefaultValues(t *testing.T) {
	// Test with invalid values should use defaults
	tool := NewSQLQueryTool(map[string]string{"test": ":memory:"}, -1, -1*time.Second)
	
	assert.Equal(t, 1000, tool.maxRows)
	assert.Equal(t, 30*time.Second, tool.queryTimeout)
}

func TestSQLQueryTool_MaxRowsLimit(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	// Create tool with very low max rows limit
	tool := NewSQLQueryTool(map[string]string{"test": dbPath}, 2, 30*time.Second)
	ctx := context.Background()

	// Try to query more rows than limit allows
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":   "query",
		"database": "test",
		"sql":      "SELECT * FROM users",
		"limit":    100, // Request more than maxRows (2)
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	rows := data["rows"].([]map[string]interface{})
	assert.LessOrEqual(t, len(rows), 2) // Should be limited to maxRows
}