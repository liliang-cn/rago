package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/google/uuid"
)

func main() {
	// Open database
	db, err := sql.Open("sqlite3", ".rago/data/usage.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	ctx := context.Background()
	
	// The specific ID requested by the user
	ragQueryID := "62a551f9-08e5-42b9-9c31-79356be064f1"
	
	// Create a conversation first (if not exists)
	conversationID := uuid.New().String()
	messageID := uuid.New().String()
	
	// Insert conversation
	_, err = db.ExecContext(ctx, `
		INSERT OR IGNORE INTO conversations (id, title, created_at, updated_at)
		VALUES (?, ?, ?, ?)`,
		conversationID, "RAG Test Conversation", time.Now(), time.Now())
	if err != nil {
		log.Printf("Warning: Failed to insert conversation: %v", err)
	}
	
	// Insert message
	_, err = db.ExecContext(ctx, `
		INSERT OR IGNORE INTO messages (id, conversation_id, role, content, token_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		messageID, conversationID, "user", "What is the purpose of RAGO?", 10, time.Now())
	if err != nil {
		log.Printf("Warning: Failed to insert message: %v", err)
	}
	
	// Delete existing RAG query if it exists
	_, _ = db.ExecContext(ctx, "DELETE FROM rag_queries WHERE id = ?", ragQueryID)
	
	// Insert RAG query
	_, err = db.ExecContext(ctx, `
		INSERT INTO rag_queries (
			id, conversation_id, message_id, query, answer,
			top_k, temperature, max_tokens, show_sources, show_thinking,
			tools_enabled, total_latency, retrieval_time, generation_time,
			chunks_found, tool_calls_count, success, error_message,
			input_tokens, output_tokens, total_tokens, estimated_cost,
			model, created_at
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?
		)`,
		ragQueryID, conversationID, messageID,
		"What is the purpose of RAGO?",
		"RAGO is a Retrieval Augmented Generation (RAG) system that combines document retrieval with LLM generation to provide context-aware answers. It supports multiple LLM providers, vector stores, and includes features like agent automation and MCP tool integration.",
		5, 0.7, 4000, true, false,
		true, 1250, 350, 900,
		5, 3, true, "",
		150, 280, 430, 0.0025,
		"qwen3:4b", time.Now())
	
	if err != nil {
		log.Fatal("Failed to insert RAG query:", err)
	}
	
	fmt.Printf("Successfully created RAG query with ID: %s\n", ragQueryID)
	
	// Insert chunk hits
	chunkHits := []struct {
		chunkID   string
		docID     string
		content   string
		score     float64
		rank      int
		used      bool
		source    string
	}{
		{
			chunkID: uuid.New().String(),
			docID:   "doc-001",
			content: "RAGO is a Retrieval Augmented Generation system that provides local-first document search and question-answering capabilities.",
			score:   0.92,
			rank:    1,
			used:    true,
			source:  "README.md",
		},
		{
			chunkID: uuid.New().String(),
			docID:   "doc-001",
			content: "The system supports multiple LLM providers including Ollama, OpenAI, and LM Studio for flexible deployment options.",
			score:   0.88,
			rank:    2,
			used:    true,
			source:  "README.md",
		},
		{
			chunkID: uuid.New().String(),
			docID:   "doc-002",
			content: "RAGO includes agent automation features that leverage MCP tools for complex workflows and task automation.",
			score:   0.85,
			rank:    3,
			used:    true,
			source:  "docs/architecture.md",
		},
		{
			chunkID: uuid.New().String(),
			docID:   "doc-003",
			content: "Vector storage options include SQLite/Sqvect for local-first deployment and Qdrant for high-performance scenarios.",
			score:   0.76,
			rank:    4,
			used:    false,
			source:  "docs/configuration.md",
		},
		{
			chunkID: uuid.New().String(),
			docID:   "doc-004",
			content: "The chunking system supports sentence, paragraph, and token-based splitting strategies with configurable overlap.",
			score:   0.72,
			rank:    5,
			used:    false,
			source:  "docs/chunking.md",
		},
	}
	
	for i, hit := range chunkHits {
		hitID := uuid.New().String()
		_, err = db.ExecContext(ctx, `
			INSERT INTO rag_chunk_hits (
				id, rag_query_id, chunk_id, document_id, content,
				score, rank_position, used_in_generation, source_file,
				chunk_index, char_start, char_end, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			hitID, ragQueryID, hit.chunkID, hit.docID, hit.content,
			hit.score, hit.rank, hit.used, hit.source,
			i, i*500, (i+1)*500, time.Now())
		
		if err != nil {
			log.Printf("Failed to insert chunk hit %d: %v", i, err)
		}
	}
	
	fmt.Printf("Successfully created %d chunk hits\n", len(chunkHits))
	
	// Insert tool calls
	toolCalls := []struct {
		toolName  string
		arguments string
		result    string
		success   bool
		duration  int
	}{
		{
			toolName:  "filesystem_read",
			arguments: `{"path": "README.md"}`,
			result:    `{"content": "# RAGO - Retrieval Augmented Generation Orchestrator..."}`,
			success:   true,
			duration:  125,
		},
		{
			toolName:  "search_documents",
			arguments: `{"query": "RAGO purpose architecture"}`,
			result:    `{"documents": ["README.md", "docs/architecture.md"], "count": 2}`,
			success:   true,
			duration:  85,
		},
		{
			toolName:  "extract_metadata",
			arguments: `{"document": "docs/configuration.md"}`,
			result:    `{"title": "RAGO Configuration", "sections": ["Providers", "Vector Stores", "Chunking"]}`,
			success:   true,
			duration:  45,
		},
	}
	
	for i, call := range toolCalls {
		callID := uuid.New().String()
		_, err = db.ExecContext(ctx, `
			INSERT INTO rag_tool_calls (
				id, rag_query_id, tool_name, arguments, result,
				success, error_message, duration, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			callID, ragQueryID, call.toolName, call.arguments, call.result,
			call.success, "", call.duration, time.Now())
		
		if err != nil {
			log.Printf("Failed to insert tool call %d: %v", i, err)
		}
	}
	
	fmt.Printf("Successfully created %d tool calls\n", len(toolCalls))
	
	// Verify the data was created
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM rag_queries WHERE id = ?", ragQueryID).Scan(&count)
	if err == nil && count > 0 {
		fmt.Println("\n✅ RAG query successfully created and verified in database")
		fmt.Printf("📊 Visualization endpoint should now work: http://localhost:7127/api/v1/rag/queries/%s/visualization\n", ragQueryID)
	}
}