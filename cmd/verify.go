package cmd

import (
	"fmt"

	"github.com/rahul/nsha/pkg/git"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify repository integrity",
	Long:  `Checks if the repository is healthy and has no corrupt objects`,
	RunE: func(cmd *cobra.Command, args []string) error {
		PrintStep(1, "Verifying repository integrity...")
		
		err := git.VerifyRepository(repoPath)
		if err != nil {
			PrintError(fmt.Sprintf("Repository has issues: %v", err))
			fmt.Println()
			PrintInfo("Run 'nsha diagnose' for detailed information")
			PrintInfo("Run 'nsha fix' to fix the issues")
			return err
		}

		PrintSuccess("Repository is healthy! No issues found.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

