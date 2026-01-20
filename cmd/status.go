package cmd

import (
	"fmt"

	"github.com/kartoza/kartoza-pg-ai/internal/config"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show application status",
	Long:  `Show the current application status including database connections and cached schemas.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Status: Not configured\n")
			fmt.Printf("Run 'kartoza-pg-ai' to configure database connections.\n")
			return
		}

		fmt.Printf("Kartoza PG AI Status\n")
		fmt.Printf("====================\n")
		fmt.Printf("Active Database: %s\n", cfg.ActiveService)
		fmt.Printf("Cached Schemas: %d\n", len(cfg.CachedSchemas))
		fmt.Printf("Query History: %d queries\n", len(cfg.QueryHistory))
	},
}
