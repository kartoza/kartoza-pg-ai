package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version number of kartoza-pg-ai.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kartoza-pg-ai version %s\n", appVersion)
	},
}
