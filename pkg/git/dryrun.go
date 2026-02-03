package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// DryRunChange represents a single change that would be made
type DryRunChange struct {
	Type        string // "reference", "tag", "commit", "tree"
	Object      string // Name of the object (e.g., "refs/heads/master", "refs/tags/v1.0")
	CurrentSHA  string // Current SHA (often null SHA)
	NewSHA      string // New SHA that will be used
	Action      string // "fix", "delete", "create", "replace"
	Description string // Human-readable description
}

// DryRunDetails holds all changes that would be made
type DryRunDetails struct {
	Changes []DryRunChange
}

// Add adds a change to the dry-run details
func (d *DryRunDetails) Add(change DryRunChange) {
	d.Changes = append(d.Changes, change)
}

// PrintSummary prints a detailed summary of all changes
func (d *DryRunDetails) PrintSummary() {
	if len(d.Changes) == 0 {
		fmt.Println("\n[DRY RUN] No changes would be made.")
		return
	}

	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║         DRY RUN - Detailed Changes Preview               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝\n")

	// Group changes by type
	byType := make(map[string][]DryRunChange)
	for _, change := range d.Changes {
		byType[change.Type] = append(byType[change.Type], change)
	}

	changeNum := 1

	// Print reference changes
	if refs, ok := byType["reference"]; ok && len(refs) > 0 {
		fmt.Printf("NULL SHA REFERENCES (%d changes):\n", len(refs))
		fmt.Println(strings.Repeat("-", 59))
		for _, change := range refs {
			fmt.Printf("\n%d. %s\n", changeNum, change.Object)
			fmt.Printf("   Current:  %s (null SHA)\n", truncateSHA(change.CurrentSHA))
			fmt.Printf("   Will fix: %s\n", truncateSHA(change.NewSHA))
			if change.Description != "" {
				fmt.Printf("   Action:   %s\n", change.Description)
			}
			changeNum++
		}
		fmt.Println()
	}

	// Print tag changes
	if tags, ok := byType["tag"]; ok && len(tags) > 0 {
		fmt.Printf("NULL SHA TAGS (%d changes):\n", len(tags))
		fmt.Println(strings.Repeat("-", 59))
		for _, change := range tags {
			fmt.Printf("\n%d. %s\n", changeNum, change.Object)
			fmt.Printf("   Current:  %s (null SHA)\n", truncateSHA(change.CurrentSHA))
			if change.Action == "delete" {
				fmt.Printf("   Will delete: No valid commit found\n")
			} else {
				fmt.Printf("   Will point to: %s\n", truncateSHA(change.NewSHA))
			}
			if change.Description != "" {
				fmt.Printf("   Details: %s\n", change.Description)
			}
			changeNum++
		}
		fmt.Println()
	}

	// Print missing commit changes
	if commits, ok := byType["missing-commit"]; ok && len(commits) > 0 {
		fmt.Printf("MISSING COMMITS (%d changes):\n", len(commits))
		fmt.Println(strings.Repeat("-", 59))
		for _, change := range commits {
			fmt.Printf("\n%d. %s\n", changeNum, change.Object)
			fmt.Printf("   Current:  %s (missing/not found)\n", truncateSHA(change.CurrentSHA))
			if change.Action == "delete" {
				fmt.Printf("   Will delete: Reference to non-existent commit\n")
			} else {
				fmt.Printf("   Will fix: %s\n", truncateSHA(change.NewSHA))
			}
			if change.Description != "" {
				fmt.Printf("   Details: %s\n", change.Description)
			}
			changeNum++
		}
		fmt.Println()
	}

	// Print tree changes
	if trees, ok := byType["tree"]; ok && len(trees) > 0 {
		fmt.Printf("CORRUPTED TREES (%d changes):\n", len(trees))
		fmt.Println(strings.Repeat("-", 59))
		for _, change := range trees {
			fmt.Printf("\n%d. Tree %s\n", changeNum, truncateSHA(change.CurrentSHA))
			fmt.Printf("   Contains null SHA entries\n")
			fmt.Printf("   Will create new tree: %s\n", truncateSHA(change.NewSHA))
			if change.Description != "" {
				fmt.Printf("   Affected files: %s\n", change.Description)
			}
			changeNum++
		}
		fmt.Println()
	}

	// Print commit replacement changes
	if commits, ok := byType["commit"]; ok && len(commits) > 0 {
		fmt.Printf("COMMIT REPLACEMENTS (%d changes):\n", len(commits))
		fmt.Println(strings.Repeat("-", 59))
		for _, change := range commits {
			fmt.Printf("\n%d. Commit %s\n", changeNum, truncateSHA(change.CurrentSHA))
			fmt.Printf("   Has broken parents or tree\n")
			fmt.Printf("   Will replace with: %s\n", truncateSHA(change.NewSHA))
			if change.Description != "" {
				fmt.Printf("   Details: %s\n", change.Description)
			}
			changeNum++
		}
		fmt.Println()
	}

	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("Total changes: %d\n", len(d.Changes))
	fmt.Printf("═══════════════════════════════════════════════════════════\n\n")
}

// truncateSHA truncates a SHA to 8 characters for display, or shows full if it's special
func truncateSHA(sha string) string {
	if sha == "" {
		return "(none)"
	}
	if strings.HasPrefix(sha, "0000000000000000000000000000000000000000") {
		return "0000000000000000000000000000000000000000 (null SHA)"
	}
	if strings.HasPrefix(sha, "ref:") {
		return sha
	}
	if len(sha) > 8 {
		return sha[:8] + "..."
	}
	return sha
}

// AnalyzeAndPopulate analyzes the repository and populates dry-run details with what would be fixed
func (d *DryRunDetails) AnalyzeAndPopulate(repoPath string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	nullSHA := "0000000000000000000000000000000000000000"

	// Find a valid commit to use as replacement
	validCommit, _ := findMostRecentValidCommit(repo)
	validRef, _ := findValidReference(repo)

	// 1. Check HEAD
	head, err := repo.Head()
	if err != nil {
		// HEAD might be broken, try to read it directly
		headPath := filepath.Join(repoPath, ".git", "HEAD")
		content, readErr := os.ReadFile(headPath)
		if readErr == nil {
			headStr := strings.TrimSpace(string(content))
			if headStr == nullSHA || strings.Contains(headStr, nullSHA) {
				d.Add(DryRunChange{
					Type:        "reference",
					Object:      "HEAD",
					CurrentSHA:  nullSHA,
					NewSHA:      validRef,
					Action:      "fix",
					Description: fmt.Sprintf("Will point to %s", validRef),
				})
			}
		}
	} else if head.Hash().String() == nullSHA {
		d.Add(DryRunChange{
			Type:        "reference",
			Object:      "HEAD",
			CurrentSHA:  nullSHA,
			NewSHA:      validRef,
			Action:      "fix",
			Description: fmt.Sprintf("Will point to %s", validRef),
		})
	}

	// 2. Check all references
	refs, err := repo.References()
	if err == nil {
		refs.ForEach(func(ref *plumbing.Reference) error {
			if ref.Hash().String() == nullSHA {
				targetSHA := validCommit
				if ref.Name().IsTag() {
					d.Add(DryRunChange{
						Type:        "tag",
						Object:      ref.Name().String(),
						CurrentSHA:  nullSHA,
						NewSHA:      targetSHA,
						Action:      "fix",
						Description: fmt.Sprintf("Will point to commit %s", targetSHA[:8]),
					})
				} else {
					d.Add(DryRunChange{
						Type:        "reference",
						Object:      ref.Name().String(),
						CurrentSHA:  nullSHA,
						NewSHA:      targetSHA,
						Action:      "fix",
						Description: fmt.Sprintf("Will point to commit %s", targetSHA[:8]),
					})
				}
			}
			return nil
		})
	}

	// 3. Check for missing commits
	refs2, _ := repo.References()
	if refs2 != nil {
		refs2.ForEach(func(ref *plumbing.Reference) error {
			_, err := repo.CommitObject(ref.Hash())
			if err != nil && !ref.Hash().IsZero() {
				d.Add(DryRunChange{
					Type:        "missing-commit",
					Object:      ref.Name().String(),
					CurrentSHA:  ref.Hash().String(),
					NewSHA:      validCommit,
					Action:      "fix",
					Description: fmt.Sprintf("Commit not found, will point to %s", validCommit[:8]),
				})
			}
			return nil
		})
	}

	// 4. Check packed-refs for null SHAs
	packedRefsPath := filepath.Join(repoPath, ".git", "packed-refs")
	if content, err := os.ReadFile(packedRefsPath); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.Contains(line, nullSHA) && !strings.HasPrefix(line, "#") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					d.Add(DryRunChange{
						Type:        "reference",
						Object:      parts[1],
						CurrentSHA:  nullSHA,
						NewSHA:      "(will be removed from packed-refs)",
						Action:      "fix",
						Description: "Will remove null SHA entry from packed-refs",
					})
				}
			}
		}
	}

	return nil
}
