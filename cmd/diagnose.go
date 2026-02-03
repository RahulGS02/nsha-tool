package cmd

import (
	"fmt"

	"github.com/rahul/nsha/pkg/git"
	"github.com/spf13/cobra"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Detect null SHA and broken tree issues",
	Long:  `Scans the repository for corrupt objects, null SHA references, and broken trees`,
	RunE: func(cmd *cobra.Command, args []string) error {
		PrintStep(1, "Scanning repository for issues...")
		
		// Run fsck
		issues, err := git.RunFsck(repoPath, verbose)
		if err != nil {
			return fmt.Errorf("fsck failed: %w", err)
		}

		if len(issues) == 0 {
			PrintSuccess("No issues found! Repository is healthy.")
			return nil
		}

		// Display issues
		PrintWarning(fmt.Sprintf("Found %d issue(s):", len(issues)))
		fmt.Println()
		
		for i, issue := range issues {
			fmt.Printf("  %d. %s\n", i+1, issue.String())
		}

		fmt.Println()
		PrintInfo("Run 'nsha fix' to automatically fix these issues")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)
}

