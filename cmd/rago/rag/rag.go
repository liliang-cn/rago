package rag

import (
	"github.com/spf13/cobra"
)

// RagCmd is the parent command for all RAG-related operations
var RagCmd = &cobra.Command{
	Use:   "rag",
	Short: "RAG (Retrieval-Augmented Generation) operations",
	Long: `Manage RAG operations including document ingestion, querying, and management.
	
Available subcommands:
  list    - List indexed documents
  ingest  - Import documents into vector database
  query   - Query knowledge base
  reset   - Clear vector database
  import  - Import knowledge base data
  export  - Export knowledge base data`,
}

func init() {
	// Add all RAG subcommands
	RagCmd.AddCommand(listCmd)
	RagCmd.AddCommand(ingestCmd)
	RagCmd.AddCommand(queryCmd)
	RagCmd.AddCommand(resetCmd)
	RagCmd.AddCommand(importCmd)
	RagCmd.AddCommand(exportCmd)
}
