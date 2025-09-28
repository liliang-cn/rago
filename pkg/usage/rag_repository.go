package usage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// RAGRepository defines methods for managing RAG query data
type RAGRepository interface {
	// RAG Query operations
	CreateRAGQuery(ctx context.Context, query *RAGQueryRecord) error
	GetRAGQuery(ctx context.Context, id string) (*RAGQueryRecord, error)
	UpdateRAGQuery(ctx context.Context, query *RAGQueryRecord) error
	ListRAGQueries(ctx context.Context, filter *RAGSearchFilter) ([]*RAGQueryRecord, error)
	
	// Chunk Hit operations
	CreateChunkHit(ctx context.Context, hit *RAGChunkHit) error
	ListChunkHits(ctx context.Context, ragQueryID string) ([]*RAGChunkHit, error)
	
	// Tool Call operations
	CreateToolCall(ctx context.Context, toolCall *RAGToolCall) error
	ListToolCalls(ctx context.Context, ragQueryID string) ([]*RAGToolCall, error)
	
	// Visualization operations
	GetRAGVisualization(ctx context.Context, ragQueryID string) (*RAGQueryVisualization, error)
	ListRAGVisualizations(ctx context.Context, filter *RAGSearchFilter) ([]*RAGQueryVisualization, error)
	
	// Analytics operations
	GetRAGAnalytics(ctx context.Context, filter *RAGSearchFilter) (*RAGAnalytics, error)
}

// InitializeRAGTables creates the necessary tables for RAG tracking
func (r *SQLiteRepository) InitializeRAGTables(ctx context.Context) error {
	queries := []string{
		// RAG queries table
		`CREATE TABLE IF NOT EXISTS rag_queries (
			id TEXT PRIMARY KEY,
			conversation_id TEXT,
			message_id TEXT,
			query TEXT NOT NULL,
			answer TEXT,
			top_k INTEGER DEFAULT 5,
			temperature REAL DEFAULT 0.7,
			max_tokens INTEGER DEFAULT 4000,
			show_sources BOOLEAN DEFAULT 0,
			show_thinking BOOLEAN DEFAULT 0,
			tools_enabled BOOLEAN DEFAULT 0,
			total_latency INTEGER DEFAULT 0,
			retrieval_time INTEGER DEFAULT 0,
			generation_time INTEGER DEFAULT 0,
			chunks_found INTEGER DEFAULT 0,
			tool_calls_count INTEGER DEFAULT 0,
			success BOOLEAN DEFAULT 0,
			error_message TEXT,
			created_at TIMESTAMP NOT NULL,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
		)`,
		
		// Chunk hits table
		`CREATE TABLE IF NOT EXISTS rag_chunk_hits (
			id TEXT PRIMARY KEY,
			rag_query_id TEXT NOT NULL,
			chunk_id TEXT NOT NULL,
			document_id TEXT NOT NULL,
			content TEXT NOT NULL,
			score REAL NOT NULL,
			rank_position INTEGER NOT NULL,
			used_in_generation BOOLEAN DEFAULT 0,
			source_file TEXT,
			chunk_index INTEGER DEFAULT 0,
			char_start INTEGER DEFAULT 0,
			char_end INTEGER DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			FOREIGN KEY (rag_query_id) REFERENCES rag_queries(id) ON DELETE CASCADE
		)`,
		
		// Tool calls table
		`CREATE TABLE IF NOT EXISTS rag_tool_calls (
			id TEXT PRIMARY KEY,
			rag_query_id TEXT NOT NULL,
			tool_name TEXT NOT NULL,
			arguments TEXT,
			result TEXT,
			success BOOLEAN DEFAULT 1,
			error_message TEXT,
			duration INTEGER DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			FOREIGN KEY (rag_query_id) REFERENCES rag_queries(id) ON DELETE CASCADE
		)`,
		
		// Indexes for better performance
		`CREATE INDEX IF NOT EXISTS idx_rag_queries_conversation_id ON rag_queries(conversation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_rag_queries_created_at ON rag_queries(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_rag_queries_success ON rag_queries(success)`,
		`CREATE INDEX IF NOT EXISTS idx_rag_chunk_hits_query_id ON rag_chunk_hits(rag_query_id)`,
		`CREATE INDEX IF NOT EXISTS idx_rag_chunk_hits_score ON rag_chunk_hits(score)`,
		`CREATE INDEX IF NOT EXISTS idx_rag_chunk_hits_rank ON rag_chunk_hits(rank_position)`,
		`CREATE INDEX IF NOT EXISTS idx_rag_tool_calls_query_id ON rag_tool_calls(rag_query_id)`,
		`CREATE INDEX IF NOT EXISTS idx_rag_tool_calls_tool_name ON rag_tool_calls(tool_name)`,
	}

	for _, query := range queries {
		if _, err := r.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute RAG table creation query: %w", err)
		}
	}

	return nil
}

// CreateRAGQuery creates a new RAG query record
func (r *SQLiteRepository) CreateRAGQuery(ctx context.Context, query *RAGQueryRecord) error {
	sql := `INSERT INTO rag_queries (
		id, conversation_id, message_id, query, answer, top_k, temperature, max_tokens,
		show_sources, show_thinking, tools_enabled, total_latency, retrieval_time,
		generation_time, chunks_found, tool_calls_count, success, error_message, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := r.db.ExecContext(ctx, sql,
		query.ID, query.ConversationID, query.MessageID, query.Query, query.Answer,
		query.TopK, query.Temperature, query.MaxTokens, query.ShowSources, query.ShowThinking,
		query.ToolsEnabled, query.TotalLatency, query.RetrievalTime, query.GenerationTime,
		query.ChunksFound, query.ToolCallsCount, query.Success, query.ErrorMessage, query.CreatedAt,
	)
	return err
}

// GetRAGQuery retrieves a RAG query by ID
func (r *SQLiteRepository) GetRAGQuery(ctx context.Context, id string) (*RAGQueryRecord, error) {
	sqlQuery := `SELECT id, conversation_id, message_id, query, answer, top_k, temperature, max_tokens,
		show_sources, show_thinking, tools_enabled, total_latency, retrieval_time,
		generation_time, chunks_found, tool_calls_count, success, error_message, created_at
		FROM rag_queries WHERE id = ?`
	
	var query RAGQueryRecord
	err := r.db.QueryRowContext(ctx, sqlQuery, id).Scan(
		&query.ID, &query.ConversationID, &query.MessageID, &query.Query, &query.Answer,
		&query.TopK, &query.Temperature, &query.MaxTokens, &query.ShowSources, &query.ShowThinking,
		&query.ToolsEnabled, &query.TotalLatency, &query.RetrievalTime, &query.GenerationTime,
		&query.ChunksFound, &query.ToolCallsCount, &query.Success, &query.ErrorMessage, &query.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("RAG query not found")
	}
	return &query, err
}

// UpdateRAGQuery updates a RAG query record
func (r *SQLiteRepository) UpdateRAGQuery(ctx context.Context, query *RAGQueryRecord) error {
	sql := `UPDATE rag_queries SET 
		answer = ?, total_latency = ?, retrieval_time = ?, generation_time = ?,
		chunks_found = ?, tool_calls_count = ?, success = ?, error_message = ?
		WHERE id = ?`
	
	_, err := r.db.ExecContext(ctx, sql,
		query.Answer, query.TotalLatency, query.RetrievalTime, query.GenerationTime,
		query.ChunksFound, query.ToolCallsCount, query.Success, query.ErrorMessage, query.ID,
	)
	return err
}

// ListRAGQueries lists RAG queries based on filter
func (r *SQLiteRepository) ListRAGQueries(ctx context.Context, filter *RAGSearchFilter) ([]*RAGQueryRecord, error) {
	sql := `SELECT id, conversation_id, message_id, query, answer, top_k, temperature, max_tokens,
		show_sources, show_thinking, tools_enabled, total_latency, retrieval_time,
		generation_time, chunks_found, tool_calls_count, success, error_message, created_at
		FROM rag_queries WHERE 1=1`
	
	args := []interface{}{}
	
	if filter.ConversationID != "" {
		sql += " AND conversation_id = ?"
		args = append(args, filter.ConversationID)
	}
	if filter.Query != "" {
		sql += " AND query LIKE ?"
		args = append(args, "%"+filter.Query+"%")
	}
	if !filter.StartTime.IsZero() {
		sql += " AND created_at >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		sql += " AND created_at <= ?"
		args = append(args, filter.EndTime)
	}
	
	sql += " ORDER BY created_at DESC"
	
	if filter.Limit > 0 {
		sql += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		sql += " OFFSET ?"
		args = append(args, filter.Offset)
	}
	
	rows, err := r.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []*RAGQueryRecord
	for rows.Next() {
		var query RAGQueryRecord
		err := rows.Scan(
			&query.ID, &query.ConversationID, &query.MessageID, &query.Query, &query.Answer,
			&query.TopK, &query.Temperature, &query.MaxTokens, &query.ShowSources, &query.ShowThinking,
			&query.ToolsEnabled, &query.TotalLatency, &query.RetrievalTime, &query.GenerationTime,
			&query.ChunksFound, &query.ToolCallsCount, &query.Success, &query.ErrorMessage, &query.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		queries = append(queries, &query)
	}

	return queries, rows.Err()
}

// CreateChunkHit creates a new chunk hit record
func (r *SQLiteRepository) CreateChunkHit(ctx context.Context, hit *RAGChunkHit) error {
	sql := `INSERT INTO rag_chunk_hits (
		id, rag_query_id, chunk_id, document_id, content, score, rank_position,
		used_in_generation, source_file, chunk_index, char_start, char_end, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := r.db.ExecContext(ctx, sql,
		hit.ID, hit.RAGQueryID, hit.ChunkID, hit.DocumentID, hit.Content, hit.Score, hit.Rank,
		hit.UsedInGeneration, hit.SourceFile, hit.ChunkIndex, hit.CharStart, hit.CharEnd, hit.CreatedAt,
	)
	return err
}

// ListChunkHits lists chunk hits for a RAG query
func (r *SQLiteRepository) ListChunkHits(ctx context.Context, ragQueryID string) ([]*RAGChunkHit, error) {
	sql := `SELECT id, rag_query_id, chunk_id, document_id, content, score, rank_position,
		used_in_generation, source_file, chunk_index, char_start, char_end, created_at
		FROM rag_chunk_hits WHERE rag_query_id = ? ORDER BY rank_position ASC`
	
	rows, err := r.db.QueryContext(ctx, sql, ragQueryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []*RAGChunkHit
	for rows.Next() {
		var hit RAGChunkHit
		err := rows.Scan(
			&hit.ID, &hit.RAGQueryID, &hit.ChunkID, &hit.DocumentID, &hit.Content, &hit.Score, &hit.Rank,
			&hit.UsedInGeneration, &hit.SourceFile, &hit.ChunkIndex, &hit.CharStart, &hit.CharEnd, &hit.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		hits = append(hits, &hit)
	}

	return hits, rows.Err()
}

// CreateToolCall creates a new tool call record
func (r *SQLiteRepository) CreateToolCall(ctx context.Context, toolCall *RAGToolCall) error {
	sql := `INSERT INTO rag_tool_calls (
		id, rag_query_id, tool_name, arguments, result, success, error_message, duration, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := r.db.ExecContext(ctx, sql,
		toolCall.ID, toolCall.RAGQueryID, toolCall.ToolName, toolCall.Arguments, toolCall.Result,
		toolCall.Success, toolCall.ErrorMessage, toolCall.Duration, toolCall.CreatedAt,
	)
	return err
}

// ListToolCalls lists tool calls for a RAG query
func (r *SQLiteRepository) ListToolCalls(ctx context.Context, ragQueryID string) ([]*RAGToolCall, error) {
	sql := `SELECT id, rag_query_id, tool_name, arguments, result, success, error_message, duration, created_at
		FROM rag_tool_calls WHERE rag_query_id = ? ORDER BY created_at ASC`
	
	rows, err := r.db.QueryContext(ctx, sql, ragQueryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []*RAGToolCall
	for rows.Next() {
		var call RAGToolCall
		err := rows.Scan(
			&call.ID, &call.RAGQueryID, &call.ToolName, &call.Arguments, &call.Result,
			&call.Success, &call.ErrorMessage, &call.Duration, &call.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		calls = append(calls, &call)
	}

	return calls, rows.Err()
}

// ListAllToolCalls lists all tool calls with filtering
func (r *SQLiteRepository) ListAllToolCalls(ctx context.Context, filter *RAGSearchFilter) ([]*RAGToolCall, error) {
	sql := `SELECT id, rag_query_id, tool_name, arguments, result, success, error_message, duration, created_at
		FROM rag_tool_calls WHERE 1=1`
	
	args := []interface{}{}
	
	if filter.Query != "" {
		sql += " AND tool_name LIKE ?"
		args = append(args, "%"+filter.Query+"%")
	}
	if !filter.StartTime.IsZero() {
		sql += " AND created_at >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		sql += " AND created_at <= ?"
		args = append(args, filter.EndTime)
	}
	
	sql += " ORDER BY created_at DESC"
	
	if filter.Limit > 0 {
		sql += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		sql += " OFFSET ?"
		args = append(args, filter.Offset)
	}
	
	rows, err := r.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []*RAGToolCall
	for rows.Next() {
		var call RAGToolCall
		err := rows.Scan(
			&call.ID, &call.RAGQueryID, &call.ToolName, &call.Arguments, &call.Result,
			&call.Success, &call.ErrorMessage, &call.Duration, &call.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		calls = append(calls, &call)
	}

	return calls, rows.Err()
}

// GetRAGVisualization gets complete visualization data for a RAG query
func (r *SQLiteRepository) GetRAGVisualization(ctx context.Context, ragQueryID string) (*RAGQueryVisualization, error) {
	// Get the query record
	query, err := r.GetRAGQuery(ctx, ragQueryID)
	if err != nil {
		return nil, err
	}
	
	// Get chunk hits
	hits, err := r.ListChunkHits(ctx, ragQueryID)
	if err != nil {
		return nil, err
	}
	
	// Get tool calls
	toolCalls, err := r.ListToolCalls(ctx, ragQueryID)
	if err != nil {
		return nil, err
	}
	
	// Convert to the format needed for metrics calculation
	hitRecords := make([]RAGChunkHit, len(hits))
	for i, hit := range hits {
		hitRecords[i] = *hit
	}
	
	// Calculate metrics
	retrievalMetrics := CalculateRetrievalMetrics(hitRecords)
	qualityMetrics := CalculateQualityMetrics(*query, hitRecords)
	
	// Convert tool calls
	toolCallRecords := make([]RAGToolCall, len(toolCalls))
	for i, call := range toolCalls {
		toolCallRecords[i] = *call
	}
	
	return &RAGQueryVisualization{
		Query:            *query,
		ChunkHits:        hitRecords,
		ToolCalls:        toolCallRecords,
		RetrievalMetrics: retrievalMetrics,
		QualityMetrics:   qualityMetrics,
	}, nil
}

// ListRAGVisualizations lists visualization data for multiple queries
func (r *SQLiteRepository) ListRAGVisualizations(ctx context.Context, filter *RAGSearchFilter) ([]*RAGQueryVisualization, error) {
	queries, err := r.ListRAGQueries(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	var visualizations []*RAGQueryVisualization
	for _, query := range queries {
		viz, err := r.GetRAGVisualization(ctx, query.ID)
		if err != nil {
			continue // Skip failed ones
		}
		visualizations = append(visualizations, viz)
	}
	
	return visualizations, nil
}

// GetRAGAnalytics gets aggregated analytics for RAG queries
func (r *SQLiteRepository) GetRAGAnalytics(ctx context.Context, filter *RAGSearchFilter) (*RAGAnalytics, error) {
	// Build the base query with filters
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	
	if filter.ConversationID != "" {
		whereClause += " AND conversation_id = ?"
		args = append(args, filter.ConversationID)
	}
	if !filter.StartTime.IsZero() {
		whereClause += " AND created_at >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		whereClause += " AND created_at <= ?"
		args = append(args, filter.EndTime)
	}
	
	// Get basic analytics with more comprehensive data
	analyticsSQL := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_queries,
			COALESCE(AVG(total_latency), 0) as avg_latency,
			COALESCE(AVG(chunks_found), 0) as avg_chunks,
			COALESCE(AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END), 0) as success_rate,
			COALESCE(SUM(CASE WHEN total_latency < 1000 THEN 1 ELSE 0 END), 0) as fast_queries,
			COALESCE(SUM(CASE WHEN total_latency >= 1000 AND total_latency <= 5000 THEN 1 ELSE 0 END), 0) as medium_queries,
			COALESCE(SUM(CASE WHEN total_latency > 5000 THEN 1 ELSE 0 END), 0) as slow_queries
		FROM rag_queries %s`, whereClause)
	
	var analytics RAGAnalytics
	err := r.db.QueryRowContext(ctx, analyticsSQL, args...).Scan(
		&analytics.TotalQueries,
		&analytics.AvgLatency,
		&analytics.AvgChunks,
		&analytics.SuccessRate,
		&analytics.FastQueries,
		&analytics.MediumQueries,
		&analytics.SlowQueries,
	)
	if err != nil {
		return nil, err
	}
	
	// Set default values for missing fields
	analytics.AvgScore = 0.0
	analytics.HighQualityQueries = 0
	analytics.MediumQualityQueries = 0
	analytics.LowQualityQueries = 0
	
	// Set time range from filter
	analytics.StartTime = filter.StartTime
	analytics.EndTime = filter.EndTime
	
	// Get top queries
	topQueriesSQL := fmt.Sprintf(`
		SELECT query, COUNT(*) as count 
		FROM rag_queries %s 
		GROUP BY query 
		ORDER BY count DESC 
		LIMIT 10`, whereClause)
	
	rows, err := r.db.QueryContext(ctx, topQueriesSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var query string
		var count int
		if err := rows.Scan(&query, &count); err != nil {
			continue
		}
		// TopQueries field doesn't exist in current RAGAnalytics struct
		// Would need to be added if this functionality is required
	}
	
	// Get popular sources - need to qualify column names for JOIN
	sourcesWhereClause := whereClause
	if sourcesWhereClause != "" {
		// Replace unqualified created_at with q.created_at
		sourcesWhereClause = strings.ReplaceAll(sourcesWhereClause, "created_at", "q.created_at")
	}
	sourcesSQL := fmt.Sprintf(`
		SELECT h.source_file, COUNT(*) as count
		FROM rag_chunk_hits h
		JOIN rag_queries q ON h.rag_query_id = q.id
		%s
		GROUP BY h.source_file
		ORDER BY count DESC
		LIMIT 10`, strings.Replace(sourcesWhereClause, "WHERE", "WHERE h.source_file IS NOT NULL AND", 1))
	
	rows, err = r.db.QueryContext(ctx, sourcesSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			continue
		}
		// PopularSources field doesn't exist in current RAGAnalytics struct
		// Would need to be added if this functionality is required
	}
	
	return &analytics, nil
}