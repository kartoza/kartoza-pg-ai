package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/kartoza/kartoza-pg-ai/internal/tui"
	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	noSplash   bool
)

// SetVersion sets the application version
func SetVersion(v string) {
	appVersion = v
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kartoza-pg-ai",
	Short: "Natural language interface to PostgreSQL databases",
	Long: `Kartoza PG AI - A beautiful TUI application for querying PostgreSQL
databases using natural language.

This tool allows you to:
  - Connect to PostgreSQL databases via pg_service.conf
  - Harvest and cache database schemas
  - Query databases using natural language
  - Maintain conversation context for follow-up queries
  - View results in beautiful formatted tables

Built with love by Kartoza.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show entry splash screen (unless --nosplash)
		if !noSplash {
			if err := tui.ShowSplashScreen(1500 * time.Millisecond); err != nil {
				fmt.Fprintf(os.Stderr, "Error showing splash: %v\n", err)
			}
		}

		// Run main TUI application
		if err := tui.RunApp(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
			os.Exit(1)
		}

		// Show exit splash screen (unless --nosplash)
		if !noSplash {
			if err := tui.ShowExitSplashScreen(800 * time.Millisecond); err != nil {
				fmt.Fprintf(os.Stderr, "Error showing exit splash: %v\n", err)
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().BoolVar(&noSplash, "nosplash", false, "Skip the splash screen animations")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(statusCmd)
}
