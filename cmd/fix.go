package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/rahul/nsha/pkg/backup"
	"github.com/rahul/nsha/pkg/git"
	"github.com/rahul/nsha/pkg/logger"
	"github.com/rahul/nsha/pkg/report"
	"github.com/spf13/cobra"
)

var (
	dryRun bool
	force  bool
	yes    bool
)

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Fix null SHA issues automatically",
	Long:  `Detects and fixes null SHA issues using git replace --graft and history rewriting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		startTime := time.Now()

		color.Cyan("\n╔═══════════════════════════════════════════════════════════╗")
		color.Cyan("║           NSHA - Null SHA Fix Process                     ║")
		color.Cyan("╚═══════════════════════════════════════════════════════════╝\n")

		// First, check if there are any issues
		PrintStep(1, "Diagnosing repository...")
		initialIssues, _ := git.RunFsck(repoPath, false)

		if len(initialIssues) == 0 {
			PrintSuccess("No issues found! Repository is healthy.")
			return nil
		}

		// In dry-run mode, just show what would be done
		var dryRunDetails *git.DryRunDetails
		if dryRun {
			dryRunDetails = &git.DryRunDetails{}
			// Analyze repository and populate dry-run details
			if err := dryRunDetails.AnalyzeAndPopulate(repoPath); err != nil {
				PrintWarning(fmt.Sprintf("Could not analyze repository for dry-run: %v", err))
			}

			color.Yellow("\n[DRY RUN MODE] - No actual changes will be made\n")
			PrintInfo(fmt.Sprintf("Found %d issue(s) that would be fixed:", len(initialIssues)))
			for i, issue := range initialIssues {
				fmt.Printf("  %d. [%s] %s: %s\n", i+1, issue.Type, issue.Object, issue.Message)
			}
		}

		// Issues found - now initialize logging and backup (skip in dry-run mode)
		var log *logger.Logger
		var backupInfo *backup.BackupInfo
		var err error

		// Initialize logger (skip in dry-run mode)
		if !dryRun {
			var logErr error
			log, logErr = logger.New(repoPath)
			if logErr != nil {
				PrintWarning(fmt.Sprintf("Could not initialize logger: %v", logErr))
				PrintWarning("Continuing without detailed logging...")
				log = nil
			} else {
				if verbose {
					PrintInfo(fmt.Sprintf("Logging to: %s", log.GetLogDir()))
				}
				log.LogStep("INITIALIZATION", "Starting NSHA fix process")
				log.LogInfo("DIAGNOSIS", fmt.Sprintf("Found %d issues requiring fixes", len(initialIssues)))
				defer func() {
					if log != nil {
						log.Close()
						if verbose {
							PrintInfo(fmt.Sprintf("Detailed logs saved to: %s", log.GetLogDir()))
						}
					}
				}()
			}

			// Create backup before any modifications
			PrintStep(2, "Creating repository backup...")
			if log != nil {
				log.LogStep("BACKUP", "Creating full repository backup with complete history")
			}

			backupInfo, err = backup.CreateBackup(repoPath, log.GetLogDir(), verbose)
			if err != nil {
				if log != nil {
					log.LogError("BACKUP", "Create backup", "Failed to create backup", err.Error())
				}
				PrintError(fmt.Sprintf("Failed to create backup: %v", err))
				PrintWarning("Do you want to continue without backup? (yes/no): ")

				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response != "yes" && response != "y" {
					return fmt.Errorf("operation cancelled: backup failed")
				}
			} else {
				if log != nil {
					log.LogInfo("BACKUP", fmt.Sprintf("Backup created successfully: %s", backupInfo.BackupPath))
				}
				PrintSuccess("Backup created successfully")

				// Verify backup
				err = backup.VerifyBackup(backupInfo, verbose)
				if err != nil {
					if log != nil {
						log.LogWarning("BACKUP", fmt.Sprintf("Backup verification failed: %v", err))
					}
					PrintWarning(fmt.Sprintf("Backup verification failed: %v", err))
					PrintWarning("Continuing anyway - backup may still be usable...")
				} else {
					if log != nil {
						log.LogInfo("BACKUP", "Backup verified successfully")
					}
				}
			}
		}

		// Step 3: Fix all types of null SHA issues
		stepNum := 3
		if dryRun {
			stepNum = 2
		}
		PrintStep(stepNum, "Fixing null SHA issues...")
		if log != nil {
			log.LogStep("FIX", "Starting null SHA fixes")
		}

		// Clean up packed-refs before starting fixes to avoid duplicates
		if !dryRun {
			if verbose {
				fmt.Println("  Cleaning up packed-refs before fixes...")
			}
			git.CleanupPackedRefs(repoPath, verbose)
		}

		totalFixCount := 0

		// 1. Fix hash-path mismatches (objects stored at null SHA paths)
		if verbose {
			fmt.Println("  Checking for hash-path mismatches...")
		}
		if log != nil {
			log.LogAction("FIX", "Check hash-path mismatches", "Scanning for objects stored at null SHA paths")
		}
		hashFixCount, hashErr := git.FixHashPathMismatch(repoPath, verbose, dryRun)
		if hashErr != nil {
			if log != nil {
				log.LogError("FIX", "Fix hash-path mismatches", "Error occurred", hashErr.Error())
			}
			if verbose {
				fmt.Printf("  Warning: Could not fix some hash-path mismatches: %v\n", hashErr)
			}
		}
		if hashFixCount > 0 {
			if log != nil {
				log.LogChange("FIX", "Fixed hash-path mismatches", "", fmt.Sprintf("%d issues", hashFixCount), "Fixed")
			}
			if dryRun {
				PrintInfo(fmt.Sprintf("[DRY RUN] Would fix %d hash-path mismatch(es)", hashFixCount))
			} else {
				PrintSuccess(fmt.Sprintf("Fixed %d hash-path mismatch(es)", hashFixCount))
			}
			totalFixCount += hashFixCount
		}

		// 2. Fix null SHA references (HEAD, branches)
		if verbose {
			fmt.Println("  Checking for null SHA references...")
		}
		if log != nil {
			log.LogAction("FIX", "Check null SHA references", "Scanning HEAD and branch references")
		}
		refFixCount, refErr := git.FixNullSHAReferences(repoPath, verbose, dryRun)
		if refErr != nil {
			if log != nil {
				log.LogError("FIX", "Fix null SHA references", "Error occurred", refErr.Error())
			}
			if verbose {
				fmt.Printf("  Warning: Could not fix some references: %v\n", refErr)
			}
		}
		if refFixCount > 0 {
			if log != nil {
				log.LogChange("FIX", "Fixed null SHA references", "", fmt.Sprintf("%d references", refFixCount), "Fixed")
			}
			if dryRun {
				PrintInfo(fmt.Sprintf("[DRY RUN] Would fix %d null SHA reference(s)", refFixCount))
			} else {
				PrintSuccess(fmt.Sprintf("Fixed %d null SHA reference(s)", refFixCount))
			}
			totalFixCount += refFixCount
		}

		// 3. Fix null SHA tags
		if verbose {
			fmt.Println("  Checking for null SHA tags...")
		}
		if log != nil {
			log.LogAction("FIX", "Check null SHA tags", "Scanning tag references")
		}
		tagFixCount, tagErr := git.FixNullSHATags(repoPath, verbose, dryRun)
		if tagErr != nil {
			if log != nil {
				log.LogError("FIX", "Fix null SHA tags", "Error occurred", tagErr.Error())
			}
			if verbose {
				fmt.Printf("  Warning: Could not fix some tags: %v\n", tagErr)
			}
		}
		if tagFixCount > 0 {
			if log != nil {
				log.LogChange("FIX", "Fixed null SHA tags", "", fmt.Sprintf("%d tags", tagFixCount), "Fixed")
			}
			if dryRun {
				PrintInfo(fmt.Sprintf("[DRY RUN] Would fix %d null SHA tag(s)", tagFixCount))
			} else {
				PrintSuccess(fmt.Sprintf("Fixed %d null SHA tag(s)", tagFixCount))
			}
			totalFixCount += tagFixCount
		}

		// 4. Fix missing commit references
		if verbose {
			fmt.Println("  Checking for missing commits...")
		}
		if log != nil {
			log.LogAction("FIX", "Check missing commits", "Scanning for references to non-existent commits")
		}
		missingFixCount, missingErr := git.FixMissingCommits(repoPath, verbose, dryRun)
		if missingErr != nil {
			if log != nil {
				log.LogError("FIX", "Fix missing commits", "Error occurred", missingErr.Error())
			}
			if verbose {
				fmt.Printf("  Warning: Could not fix some missing commits: %v\n", missingErr)
			}
		}
		if missingFixCount > 0 {
			if log != nil {
				log.LogChange("FIX", "Fixed missing commits", "", fmt.Sprintf("%d references", missingFixCount), "Fixed")
			}
			if dryRun {
				PrintInfo(fmt.Sprintf("[DRY RUN] Would fix %d missing commit reference(s)", missingFixCount))
			} else {
				PrintSuccess(fmt.Sprintf("Fixed %d missing commit reference(s)", missingFixCount))
			}
			totalFixCount += missingFixCount
		}

		// 5. Fix tree objects with null SHA entries
		if verbose {
			fmt.Println("  Checking for corrupted tree objects...")
		}
		if log != nil {
			log.LogAction("FIX", "Check tree corruption", "Scanning for tree objects with null SHA entries")
		}
		treeFixCount, treeErr := git.FixTreeObjectsWithNullSHA(repoPath, verbose, dryRun)
		if treeErr != nil {
			if log != nil {
				log.LogError("FIX", "Fix tree corruption", "Error occurred", treeErr.Error())
			}
			if verbose {
				fmt.Printf("  Warning: Could not fix some tree objects: %v\n", treeErr)
			}
		}
		if treeFixCount > 0 {
			if log != nil {
				log.LogChange("FIX", "Fixed tree corruption", "", fmt.Sprintf("%d trees", treeFixCount), "Fixed")
			}
			if dryRun {
				PrintInfo(fmt.Sprintf("[DRY RUN] Would fix %d corrupted tree object(s)", treeFixCount))
			} else {
				PrintSuccess(fmt.Sprintf("Fixed %d corrupted tree object(s)", treeFixCount))
			}
			totalFixCount += treeFixCount
		}

		// Now check for bad commits that need history rewriting
		if verbose {
			fmt.Println("  Checking for commits that need history rewriting...")
		}
		if log != nil {
			log.LogAction("FIX", "Check bad commits", "Scanning for commits requiring history rewriting")
		}
		badCommits, err := git.FindBadCommits(repoPath)
		if err != nil {
			if log != nil {
				log.LogError("FIX", "Find bad commits", "Error occurred", err.Error())
			}
			return fmt.Errorf("diagnosis failed: %w", err)
		}
		if log != nil {
			log.LogInfo("FIX", fmt.Sprintf("Found %d bad commits requiring history rewriting", len(badCommits)))
		}

		if len(badCommits) == 0 && totalFixCount > 0 {
			// Only references/paths/tags were fixed, no commits to fix
			if dryRun {
				PrintInfo(fmt.Sprintf("[DRY RUN] Would fix %d issue(s)!", totalFixCount))

				// Print detailed dry-run summary
				if dryRunDetails != nil && len(dryRunDetails.Changes) > 0 {
					dryRunDetails.PrintSummary()
				}
			} else {
				PrintSuccess(fmt.Sprintf("Fixed %d issue(s)!", totalFixCount))

				// Run garbage collection to clean up orphaned objects
				if verbose {
					fmt.Println("  Running garbage collection to clean up orphaned objects...")
				}
				if log != nil {
					log.LogStep("CLEANUP", "Running garbage collection")
				}
				gcErr := git.RunGarbageCollection(repoPath, verbose)
				if gcErr != nil {
					if verbose {
						fmt.Printf("  Warning: Garbage collection failed: %v\n", gcErr)
					}
					if log != nil {
						log.LogWarning("CLEANUP", fmt.Sprintf("Garbage collection failed: %v", gcErr))
					}
				} else {
					if log != nil {
						log.LogInfo("CLEANUP", "Garbage collection completed")
					}
				}
			}

			// Verify the fix
			stepNum := 2
			if dryRun {
				stepNum = 3
			}
			PrintStep(stepNum, "Verifying repository integrity...")
			if log != nil {
				log.LogStep("VERIFICATION", "Verifying repository integrity")
			}
			err = git.VerifyRepository(repoPath)
			if err != nil {
				if log != nil {
					log.LogWarning("VERIFICATION", fmt.Sprintf("Verification found issues: %v", err))
				}
				PrintWarning("Verification found remaining issues:")
				fmt.Printf("  %v\n", err)
				fmt.Println()

				// Show different message for dry-run vs actual fix
				if dryRun {
					color.Cyan("\n[INFO] [DRY RUN] Issues remain because no changes were made.")
					color.Cyan("   Run without --dry-run to apply fixes: nsha fix --repo <path>")
				} else {
					PrintInfo("Some issues may require manual intervention or running 'nsha fix' again")
				}
			} else {
				if log != nil {
					log.LogInfo("VERIFICATION", "Repository verified successfully")
				}
				PrintSuccess("Repository verified - all issues fixed!")
			}

			// Generate reports
			if log != nil {
				log.LogStep("REPORTING", "Generating detailed reports")
				finalIssues, _ := git.RunFsck(repoPath, false)

				reportData := &report.ReportData{
					RepoPath:      repoPath,
					StartTime:     startTime,
					EndTime:       time.Now(),
					InitialIssues: initialIssues,
					FinalIssues:   finalIssues,
					Operations:    log.GetOperations(),
					BackupPath:    "",
					Success:       len(finalIssues) == 0,
				}

				if backupInfo != nil {
					reportData.BackupPath = backupInfo.BackupPath
				}

				err = report.GenerateReport(reportData, log.GetLogDir())
				if err != nil {
					log.LogWarning("REPORTING", fmt.Sprintf("Could not generate reports: %v", err))
					PrintWarning(fmt.Sprintf("Could not generate reports: %v", err))
				} else {
					log.LogInfo("REPORTING", "Reports generated successfully")
					PrintInfo(fmt.Sprintf("Reports saved to: %s", log.GetLogDir()))
				}
			}

			return nil
		}

		fmt.Printf("\n  Found %d bad commit(s):\n", len(badCommits))
		for i, commit := range badCommits {
			fmt.Printf("    %d. %s\n", i+1, commit.String())
		}

		// Confirmation prompt
		if !yes && !dryRun {
			fmt.Println()
			PrintWarning("This operation will rewrite Git history!")
			fmt.Print("\n  Do you want to continue? (yes/no): ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "yes" && response != "y" {
				PrintInfo("Operation cancelled by user")
				return nil
			}
		}

		// Step 2: Create empty tree
		PrintStep(2, "Creating empty tree object...")
		if log != nil {
			log.LogStep("REWRITE", "Creating empty tree object")
		}
		emptyTree, err := git.CreateEmptyTree(repoPath)
		if err != nil {
			if log != nil {
				log.LogError("REWRITE", "Create empty tree", "Failed to create empty tree", err.Error())
			}
			return fmt.Errorf("failed to create empty tree: %w", err)
		}

		if verbose {
			fmt.Printf("  Empty tree hash: %s\n", emptyTree)
		}
		if log != nil {
			log.LogInfo("REWRITE", fmt.Sprintf("Empty tree created: %s", emptyTree))
		}
		PrintSuccess("Empty tree created")

		// Step 3: Replace commits
		PrintStep(3, "Replacing broken commits...")
		if log != nil {
			log.LogStep("REWRITE", fmt.Sprintf("Replacing %d broken commits", len(badCommits)))
		}
		for i, commit := range badCommits {
			if dryRun {
				fmt.Printf("  [DRY RUN] Would replace: %s\n", commit.Hash[:8])
				if log != nil {
					log.LogInfo("REWRITE", fmt.Sprintf("[DRY RUN] Would replace commit: %s", commit.Hash))
				}
			} else {
				err := git.ReplaceCommit(repoPath, commit)
				if err != nil {
					if log != nil {
						log.LogError("REWRITE", "Replace commit", commit.Hash, err.Error())
					}
					PrintError(fmt.Sprintf("Failed to replace %s: %v", commit.Hash[:8], err))
					continue
				}
				if log != nil {
					log.LogChange("REWRITE", "Replaced commit", commit.Hash, "Broken commit", "Replaced with valid commit")
				}
				fmt.Printf("  ✓ Replaced %d/%d: %s\n", i+1, len(badCommits), commit.Hash[:8])
			}
		}

		if !dryRun {
			PrintSuccess("All commits replaced")
		}

		// Step 4: Rewrite history
		if !dryRun {
			PrintStep(4, "Rewriting history (this may take a while)...")
			if log != nil {
				log.LogStep("REWRITE", "Rewriting repository history with git filter-repo")
			}
			err = git.FilterRepo(repoPath, force)
			if err != nil {
				if log != nil {
					log.LogError("REWRITE", "Filter repository", "History rewrite failed", err.Error())
				}
				return fmt.Errorf("history rewrite failed: %w", err)
			}
			if log != nil {
				log.LogInfo("REWRITE", "History rewritten successfully")
			}
			PrintSuccess("History rewritten successfully")

			// Step 5: Cleanup
			PrintStep(5, "Cleaning up replace references...")
			if log != nil {
				log.LogStep("CLEANUP", "Cleaning up replace references")
			}
			err = git.CleanupReplaceRefs(repoPath)
			if err != nil {
				if log != nil {
					log.LogError("CLEANUP", "Cleanup replace refs", "Cleanup failed", err.Error())
				}
				return fmt.Errorf("cleanup failed: %w", err)
			}
			if log != nil {
				log.LogInfo("CLEANUP", "Replace references cleaned up")
			}
			PrintSuccess("Cleanup complete")

			// Step 6: Verify
			PrintStep(6, "Verifying repository integrity...")
			if log != nil {
				log.LogStep("VERIFICATION", "Verifying repository integrity")
			}
			err = git.VerifyRepository(repoPath)
			if err != nil {
				if log != nil {
					log.LogWarning("VERIFICATION", fmt.Sprintf("Verification found issues: %v", err))
				}
				PrintWarning("Verification found issues:")
				fmt.Printf("  %v\n", err)
				fmt.Println()
				PrintInfo("You may need to run 'nsha fix' again")
			} else {
				if log != nil {
					log.LogInfo("VERIFICATION", "Repository verified successfully")
				}
				PrintSuccess("Repository verified - all issues fixed!")
			}
		}

		// Generate comprehensive reports
		if log != nil && !dryRun {
			log.LogStep("REPORTING", "Generating detailed reports")
			finalIssues, _ := git.RunFsck(repoPath, false)

			reportData := &report.ReportData{
				RepoPath:      repoPath,
				StartTime:     startTime,
				EndTime:       time.Now(),
				InitialIssues: initialIssues,
				FinalIssues:   finalIssues,
				Operations:    log.GetOperations(),
				BackupPath:    "",
				Success:       len(finalIssues) == 0,
			}

			if backupInfo != nil {
				reportData.BackupPath = backupInfo.BackupPath
			}

			err = report.GenerateReport(reportData, log.GetLogDir())
			if err != nil {
				log.LogWarning("REPORTING", fmt.Sprintf("Could not generate reports: %v", err))
				PrintWarning(fmt.Sprintf("Could not generate reports: %v", err))
			} else {
				log.LogInfo("REPORTING", "Reports generated successfully")
				PrintInfo(fmt.Sprintf("\n📊 Detailed reports saved to: %s", log.GetLogDir()))
			}
		}

		// Final message
		fmt.Println()
		color.Green("╔═══════════════════════════════════════════════════════════╗")
		if dryRun {
			color.Green("║              DRY RUN COMPLETE                             ║")
			color.Green("║  Run without --dry-run to apply changes                  ║")
		} else {
			color.Green("║              FIX COMPLETE!                                ║")
			color.Green("║                                                           ║")
			color.Green("║  Next steps:                                              ║")
			color.Green("║  1. Review the changes with: git log                      ║")
			color.Green("║  2. Push to remote: git push origin --force --all         ║")
			color.Green("║  3. Push tags: git push origin --force --tags             ║")
		}
		color.Green("╚═══════════════════════════════════════════════════════════╝\n")

		return nil
	},
}

func init() {
	fixCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	fixCmd.Flags().BoolVarP(&force, "force", "f", false, "Force history rewrite even if there are warnings")
	fixCmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	rootCmd.AddCommand(fixCmd)
}
