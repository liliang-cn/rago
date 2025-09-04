package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/liliang-cn/rago/v2/pkg/store"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List indexed documents",
	Long:  `Display all documents imported into the vector database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vectorStore, err := store.NewSQLiteStore(
			cfg.Sqvect.DBPath,
		)
		if err != nil {
			return fmt.Errorf("failed to create vector store: %w", err)
		}
		defer func() {
			if closeErr := vectorStore.Close(); closeErr != nil {
				fmt.Printf("Warning: failed to close vector store: %v\n", closeErr)
			}
		}()

		ctx := context.Background()
		documents, err := vectorStore.List(ctx)
		if err != nil {
			return fmt.Errorf("failed to list documents: %w", err)
		}

		if len(documents) == 0 {
			fmt.Println("No documents found in the knowledge base.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintf(w, "ID\tPATH\tURL\tCREATED\n"); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
		if _, err := fmt.Fprintf(w, "---\t----\t---\t-------\n"); err != nil {
			return fmt.Errorf("failed to write separator: %w", err)
		}

		for _, doc := range documents {
			path := doc.Path
			if path == "" {
				path = "-"
			}

			url := doc.URL
			if url == "" {
				url = "-"
			}

			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				truncateText(doc.ID, 36),
				truncateText(path, 50),
				truncateText(url, 50),
				doc.Created.Format("2006-01-02 15:04:05"),
			); err != nil {
				return fmt.Errorf("failed to write document row: %w", err)
			}
		}

		if err := w.Flush(); err != nil {
			return fmt.Errorf("failed to flush output: %w", err)
		}

		fmt.Printf("\nTotal: %d documents\n", len(documents))
		return nil
	},
}
