package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	repoPath string
	verbose  bool
)

var rootCmd = &cobra.Command{
	Use:   "nsha",
	Short: "Fix null SHA and broken tree issues in Git repositories",
	Long: color.CyanString(`
╔═══════════════════════════════════════════════════════════╗
║                    NSHA - Null SHA Fixer                  ║
║                                                           ║
║  A powerful CLI tool to detect and fix null SHA          ║
║  (0000000...) and broken tree object issues in           ║
║  Git repositories.                                        ║
╚═══════════════════════════════════════════════════════════╝
`),
	Version:           "1.3.0",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&repoPath, "repo", "r", ".", "Path to Git repository")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

// Helper functions for colored output
func PrintSuccess(msg string) {
	color.Green("[SUCCESS] " + msg)
}

func PrintError(msg string) {
	color.Red("[ERROR] " + msg)
}

func PrintWarning(msg string) {
	color.Yellow("[WARNING] " + msg)
}

func PrintInfo(msg string) {
	color.Cyan("[INFO] " + msg)
}

func PrintStep(step int, msg string) {
	color.Cyan(fmt.Sprintf("\n[STEP %d] %s", step, msg))
}

func ExitWithError(msg string, err error) {
	if err != nil {
		PrintError(fmt.Sprintf("%s: %v", msg, err))
	} else {
		PrintError(msg)
	}
	os.Exit(1)
}
