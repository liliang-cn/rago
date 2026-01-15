package rag

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/rag/store"
	"github.com/spf13/cobra"
)

var (
	force bool
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Clear vector database",
	Long:  `Delete all indexed documents and vector data. This operation cannot be undone!`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !force {
			fmt.Print("Are you sure you want to reset the database? This cannot be undone! (y/N): ")
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() {
				return fmt.Errorf("failed to read input")
			}

			input := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if input != "y" && input != "yes" {
				fmt.Println("Reset cancelled.")
				return nil
			}
		}

		// Initialize vector store based on configuration
		var vectorStore domain.VectorStore
		var err error
		
		if Cfg.VectorStore != nil && Cfg.VectorStore.Type != "" {
			// Use configured vector store
			storeConfig := store.StoreConfig{
				Type:       Cfg.VectorStore.Type,
				Parameters: Cfg.VectorStore.Parameters,
			}
			vectorStore, err = store.NewVectorStore(storeConfig)
			if err != nil {
				return fmt.Errorf("failed to create vector store: %w", err)
			}
		} else {
			// Default to SQLite
			vectorStore, err = store.NewSQLiteStore(Cfg.Sqvect.DBPath, Cfg.Sqvect.IndexType)
			if err != nil {
				return fmt.Errorf("failed to create vector store: %w", err)
			}
		}
		
		// Close the store when done
		defer func() {
			if closer, ok := vectorStore.(interface{ Close() error }); ok {
				if err := closer.Close(); err != nil {
					fmt.Printf("Warning: failed to close vector store: %v\n", err)
				}
			}
		}()

		ctx := context.Background()
		if err := vectorStore.Reset(ctx); err != nil {
			return fmt.Errorf("failed to reset vector store: %w", err)
		}

		fmt.Println("Database has been reset successfully.")
		return nil
	},
}

func init() {
	resetCmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
}
