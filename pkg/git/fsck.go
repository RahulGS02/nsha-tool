package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// RunFsck performs a full repository check similar to git fsck
func RunFsck(repoPath string, verbose bool) ([]Issue, error) {
	var issues []Issue

	// First, run the actual git fsck command to catch hash-path mismatches and other issues
	cmd := exec.Command("git", "fsck", "--full")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()

	// Parse git fsck output
	if len(output) > 0 {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Parse different types of errors
			if strings.Contains(line, "hash-path mismatch") {
				// Extract the hash and path
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					hash := strings.TrimSpace(parts[1])
					hash = strings.Split(hash, " ")[0]

					// Check if it's stored at null SHA path
					if strings.Contains(line, "objects/00/0000000000000000000000000000000000000") {
						issues = append(issues, Issue{
							Type:    IssueTypeNullSHA,
							Object:  hash,
							Message: "Object stored at null SHA path (hash-path mismatch)",
						})
					}
				}
			} else if strings.Contains(line, "null sha1") || strings.Contains(line, "null SHA") {
				issues = append(issues, Issue{
					Type:    IssueTypeNullSHA,
					Object:  "",
					Message: line,
				})
			} else if strings.Contains(line, "missing") {
				issues = append(issues, Issue{
					Type:    IssueTypeMissingCommit,
					Object:  "",
					Message: line,
				})
			} else if strings.HasPrefix(line, "error:") || strings.HasPrefix(line, "warning:") {
				// Generic error/warning
				if verbose {
					fmt.Printf("  Git fsck: %s\n", line)
				}
			}
		}
	}

	// Now also check using go-git for additional checks
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return issues, nil // Return what we found from git fsck
	}

	// Check all references
	refs, err := repo.References()
	if err != nil {
		return issues, nil
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if verbose {
			fmt.Printf("Checking ref: %s\n", ref.Name())
		}

		// Skip symbolic references (like HEAD when it points to a branch)
		// They will be checked through their target
		if ref.Type() == plumbing.SymbolicReference {
			if verbose {
				fmt.Printf("  Skipping symbolic reference: %s -> %s\n", ref.Name(), ref.Target())
			}
			return nil
		}

		// Check for null SHA
		hashStr := ref.Hash().String()
		if ref.Hash().IsZero() || hashStr == "0000000000000000000000000000000000000000" {
			issues = append(issues, Issue{
				Type:    IssueTypeNullSHA,
				Object:  ref.Name().String(),
				Message: fmt.Sprintf("Reference has null SHA"),
			})
			return nil
		}

		// Try to get the commit
		commit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			issues = append(issues, Issue{
				Type:    IssueTypeMissingCommit,
				Object:  ref.Hash().String(),
				Message: fmt.Sprintf("Cannot read commit: %v", err),
			})
			return nil
		}

		// Check tree
		_, err = commit.Tree()
		if err != nil {
			issues = append(issues, Issue{
				Type:    IssueTypeMissingTree,
				Object:  commit.TreeHash.String(),
				Commit:  commit.Hash.String(),
				Message: fmt.Sprintf("Commit references missing tree"),
			})
		}

		// Check parents
		for _, parentHash := range commit.ParentHashes {
			if parentHash.IsZero() || parentHash.String() == "0000000000000000000000000000000000000000" {
				issues = append(issues, Issue{
					Type:    IssueTypeBrokenParent,
					Object:  commit.Hash.String(),
					Message: fmt.Sprintf("Commit has null parent SHA"),
				})
			}
		}

		return nil
	})

	if err != nil {
		return issues, nil
	}

	return issues, nil
}

// FindBadCommits identifies all commits that need to be fixed
func FindBadCommits(repoPath string) ([]BadCommit, error) {
	issues, err := RunFsck(repoPath, false)
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	badCommitsMap := make(map[string]*BadCommit)

	for _, issue := range issues {
		// Handle ALL issue types including null SHA references
		if issue.Type == IssueTypeMissingTree || issue.Type == IssueTypeBrokenParent || issue.Type == IssueTypeNullSHA {
			commitHash := issue.Commit
			if commitHash == "" {
				commitHash = issue.Object
			}

			// Skip if this is a pure reference issue (no commit to fix)
			if commitHash == "" || commitHash == "0000000000000000000000000000000000000000" {
				continue
			}

			if _, exists := badCommitsMap[commitHash]; !exists {
				commit, err := repo.CommitObject(plumbing.NewHash(commitHash))
				if err != nil {
					continue
				}

				bc := &BadCommit{
					Hash:     commitHash,
					TreeHash: commit.TreeHash.String(),
					Message:  commit.Message,
					IsRoot:   len(commit.ParentHashes) == 0,
				}

				if commit.Author.Name != "" {
					bc.Author = commit.Author.Name
					bc.AuthorEmail = commit.Author.Email
					bc.AuthorDate = commit.Author.When.Format("2006-01-02 15:04:05 -0700")
				}

				if commit.Committer.Name != "" {
					bc.Committer = commit.Committer.Name
					bc.CommitterEmail = commit.Committer.Email
					bc.CommitterDate = commit.Committer.When.Format("2006-01-02 15:04:05 -0700")
				}

				if len(commit.ParentHashes) > 0 && !commit.ParentHashes[0].IsZero() {
					bc.ParentHash = commit.ParentHashes[0].String()
				}

				badCommitsMap[commitHash] = bc
			}
		}
	}

	var badCommits []BadCommit
	for _, bc := range badCommitsMap {
		badCommits = append(badCommits, *bc)
	}

	return badCommits, nil
}

// FixHashPathMismatch fixes objects stored at wrong paths (null SHA paths)
func FixHashPathMismatch(repoPath string, verbose bool, dryRun bool) (int, error) {
	fixedCount := 0

	// Run git fsck to find hash-path mismatches
	cmd := exec.Command("git", "fsck", "--full")
	cmd.Dir = repoPath
	output, _ := cmd.CombinedOutput()

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if !strings.Contains(line, "hash-path mismatch") {
			continue
		}

		// Parse: "error: <actual-hash>: hash-path mismatch, found at: .git/objects/00/0000..."
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}

		actualHash := strings.TrimSpace(parts[1])
		wrongPath := ""

		// Find the part with "found at"
		for i, part := range parts {
			if strings.Contains(part, "found at") {
				// The path is in the next part after "found at"
				if i+1 < len(parts) {
					wrongPath = strings.TrimSpace(parts[i+1])
				} else {
					// Path might be in the same part after "found at"
					afterFoundAt := strings.Split(part, "found at")
					if len(afterFoundAt) > 1 {
						wrongPath = strings.TrimSpace(afterFoundAt[1])
					}
				}
				break
			}
		}

		if actualHash == "" || wrongPath == "" {
			continue
		}

		if verbose {
			if dryRun {
				fmt.Printf("  [DRY RUN] Would fix hash-path mismatch: %s at %s\n", actualHash[:8], wrongPath)
			} else {
				fmt.Printf("  Found hash-path mismatch: %s at %s\n", actualHash[:8], wrongPath)
			}
		}

		if dryRun {
			fixedCount++
			continue
		}

		// Calculate correct path
		correctPath := filepath.Join(repoPath, ".git", "objects", actualHash[:2], actualHash[2:])
		wrongFullPath := filepath.Join(repoPath, wrongPath)

		// Create directory for correct path
		correctDir := filepath.Dir(correctPath)
		if err := os.MkdirAll(correctDir, 0755); err != nil {
			if verbose {
				fmt.Printf("  Failed to create directory %s: %v\n", correctDir, err)
			}
			continue
		}

		// Move the object to correct location
		if err := os.Rename(wrongFullPath, correctPath); err != nil {
			// If rename fails, try copy and delete
			content, readErr := os.ReadFile(wrongFullPath)
			if readErr == nil {
				if writeErr := os.WriteFile(correctPath, content, 0444); writeErr == nil {
					os.Remove(wrongFullPath)
					if verbose {
						fmt.Printf("  Moved object %s to correct path\n", actualHash[:8])
					}
					fixedCount++
				}
			}
		} else {
			if verbose {
				fmt.Printf("  Moved object %s to correct path\n", actualHash[:8])
			}
			fixedCount++
		}
	}

	if !dryRun {
		// Clean up empty null SHA directories
		nullDirs := []string{
			filepath.Join(repoPath, ".git", "objects", "00"),
		}

		for _, dir := range nullDirs {
			if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
				os.Remove(dir)
				if verbose {
					fmt.Printf("  Removed empty directory: %s\n", dir)
				}
			}
		}
	}

	return fixedCount, nil
}

// FixNullSHAReferences fixes null SHA in references (HEAD, branches, tags)
func FixNullSHAReferences(repoPath string, verbose bool, dryRun bool) (int, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open repository: %w", err)
	}

	fixedCount := 0
	nullSHA := "0000000000000000000000000000000000000000"

	// 1. Check and fix HEAD reference
	head, err := repo.Head()
	if err != nil {
		// HEAD might be broken, try to read it directly
		headPath := filepath.Join(repoPath, ".git", "HEAD")
		content, readErr := os.ReadFile(headPath)
		if readErr == nil {
			headStr := strings.TrimSpace(string(content))

			// Check if HEAD contains null SHA
			if headStr == nullSHA || strings.Contains(headStr, nullSHA) {
				if verbose {
					if dryRun {
						fmt.Println("  [DRY RUN] Would fix null SHA in HEAD reference")
					} else {
						fmt.Println("  Found null SHA in HEAD reference")
					}
				}

				if dryRun {
					fixedCount++
				} else {
					// Try to find a valid branch to point to
					validRef, findErr := findValidReference(repo)
					if findErr == nil && validRef != "" {
						// Update HEAD to point to valid branch
						newContent := fmt.Sprintf("ref: %s\n", validRef)
						if writeErr := os.WriteFile(headPath, []byte(newContent), 0644); writeErr == nil {
							if verbose {
								fmt.Printf("  Fixed HEAD -> %s\n", validRef)
							}
							fixedCount++
						}
					}
				}
			}
		}
	} else if head.Hash().String() == nullSHA {
		if verbose {
			if dryRun {
				fmt.Println("  [DRY RUN] Would fix null SHA in HEAD reference")
			} else {
				fmt.Println("  Found null SHA in HEAD reference")
			}
		}

		if dryRun {
			fixedCount++
		} else {
			// Try to find a valid branch
			validRef, findErr := findValidReference(repo)
			if findErr == nil && validRef != "" {
				headPath := filepath.Join(repoPath, ".git", "HEAD")
				newContent := fmt.Sprintf("ref: %s\n", validRef)
				if writeErr := os.WriteFile(headPath, []byte(newContent), 0644); writeErr == nil {
					if verbose {
						fmt.Printf("  Fixed HEAD -> %s\n", validRef)
					}
					fixedCount++
				}
			}
		}
	}

	// 2. Check and fix branch references
	refs, err := repo.References()
	if err == nil {
		err = refs.ForEach(func(ref *plumbing.Reference) error {
			if ref.Hash().String() == nullSHA {
				if verbose {
					if dryRun {
						fmt.Printf("  [DRY RUN] Would fix null SHA in reference: %s\n", ref.Name())
					} else {
						fmt.Printf("  Found null SHA in reference: %s\n", ref.Name())
					}
				}

				if dryRun {
					fixedCount++
				} else {
					// For branches with null SHA, try to find a valid commit
					if ref.Name().IsBranch() {
						validCommit, findErr := findMostRecentValidCommit(repo)
						if findErr == nil && validCommit != "" {
							// Update the branch reference
							refPath := filepath.Join(repoPath, ".git", ref.Name().String())
							if writeErr := os.WriteFile(refPath, []byte(validCommit+"\n"), 0644); writeErr == nil {
								if verbose {
									fmt.Printf("  Fixed branch %s -> %s\n", ref.Name().Short(), validCommit[:8])
								}
								fixedCount++
							}
						}
					}
				}
			}
			return nil
		})
	}

	// 3. Check and fix packed-refs
	packedRefsPath := filepath.Join(repoPath, ".git", "packed-refs")
	if content, err := os.ReadFile(packedRefsPath); err == nil {
		lines := strings.Split(string(content), "\n")
		modified := false
		var newLines []string
		seenRefs := make(map[string]bool) // Track seen references to avoid duplicates

		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)

			// Skip empty lines in dry-run mode
			if dryRun && trimmedLine == "" {
				continue
			}

			// Skip lines with null SHA
			if strings.Contains(line, nullSHA) {
				if verbose {
					if dryRun {
						fmt.Printf("  [DRY RUN] Would remove null SHA in packed-refs: %s\n", line)
					} else {
						fmt.Printf("  Found null SHA in packed-refs: %s\n", line)
					}
				}
				modified = true
				fixedCount++
				continue // Skip this line (both in dry-run and actual fix)
			}

			// Extract reference name to check for duplicates
			if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "^") {
				parts := strings.Fields(trimmedLine)
				if len(parts) >= 2 {
					refName := parts[1]
					if seenRefs[refName] {
						// Duplicate reference found
						if verbose {
							if dryRun {
								fmt.Printf("  [DRY RUN] Would remove duplicate in packed-refs: %s\n", line)
							} else {
								fmt.Printf("  Found duplicate in packed-refs: %s\n", line)
							}
						}
						modified = true
						continue // Skip duplicate
					}
					seenRefs[refName] = true
				}
			}

			if !dryRun {
				newLines = append(newLines, line)
			}
		}

		if modified && !dryRun {
			newContent := strings.Join(newLines, "\n")
			if err := os.WriteFile(packedRefsPath, []byte(newContent), 0644); err == nil {
				if verbose {
					fmt.Println("  Fixed packed-refs file")
				}
			}
		}
	}

	return fixedCount, nil
}

// findValidReference finds a valid branch reference to point HEAD to
func findValidReference(repo *git.Repository) (string, error) {
	// Try common branch names first
	commonBranches := []string{"refs/heads/main", "refs/heads/master", "refs/heads/develop"}

	for _, branchName := range commonBranches {
		ref, err := repo.Reference(plumbing.ReferenceName(branchName), true)
		if err == nil && ref.Hash().String() != "0000000000000000000000000000000000000000" {
			return branchName, nil
		}
	}

	// If no common branch found, find any valid branch
	refs, err := repo.References()
	if err != nil {
		return "", err
	}

	var validRef string
	refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() && ref.Hash().String() != "0000000000000000000000000000000000000000" {
			validRef = ref.Name().String()
			return fmt.Errorf("found") // Stop iteration
		}
		return nil
	})

	if validRef != "" {
		return validRef, nil
	}

	return "", fmt.Errorf("no valid branch found")
}

// findMostRecentValidCommit finds the most recent valid commit in the repository
func findMostRecentValidCommit(repo *git.Repository) (string, error) {
	// Try to get commits from all branches
	refs, err := repo.References()
	if err != nil {
		return "", err
	}

	var mostRecentCommit *object.Commit

	refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() && ref.Hash().String() != "0000000000000000000000000000000000000000" {
			commit, err := repo.CommitObject(ref.Hash())
			if err == nil {
				if mostRecentCommit == nil || commit.Committer.When.After(mostRecentCommit.Committer.When) {
					mostRecentCommit = commit
				}
			}
		}
		return nil
	})

	if mostRecentCommit != nil {
		return mostRecentCommit.Hash.String(), nil
	}

	return "", fmt.Errorf("no valid commit found")
}

// VerifyRepository checks if the repository is healthy
func VerifyRepository(repoPath string) error {
	issues, err := RunFsck(repoPath, false)
	if err != nil {
		return err
	}

	if len(issues) > 0 {
		var msgs []string
		for _, issue := range issues {
			msgs = append(msgs, issue.String())
		}
		return fmt.Errorf("found %d issue(s):\n%s", len(issues), strings.Join(msgs, "\n"))
	}

	return nil
}

// FixNullSHATags fixes all tags that point to null SHA
func FixNullSHATags(repoPath string, verbose bool, dryRun bool) (int, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open repository: %w", err)
	}

	fixedCount := 0
	nullSHA := "0000000000000000000000000000000000000000"

	// Get all tag references
	refs, err := repo.References()
	if err != nil {
		return 0, err
	}

	var tagsToFix []string

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsTag() && ref.Hash().String() == nullSHA {
			tagsToFix = append(tagsToFix, ref.Name().String())
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	// Fix each tag
	for _, tagName := range tagsToFix {
		if verbose {
			if dryRun {
				fmt.Printf("  [DRY RUN] Would fix null SHA tag: %s\n", tagName)
			} else {
				fmt.Printf("  Found null SHA tag: %s\n", tagName)
			}
		}

		if dryRun {
			fixedCount++
			continue
		}

		// Option 1: Try to find a valid commit to point the tag to
		validCommit, findErr := findMostRecentValidCommit(repo)
		if findErr == nil && validCommit != "" {
			// Update the tag to point to valid commit
			tagPath := filepath.Join(repoPath, ".git", tagName)
			if writeErr := os.WriteFile(tagPath, []byte(validCommit+"\n"), 0644); writeErr == nil {
				if verbose {
					fmt.Printf("  Fixed tag %s -> %s\n", filepath.Base(tagName), validCommit[:8])
				}
				fixedCount++
			} else {
				// If writing fails, try to delete the tag
				if verbose {
					fmt.Printf("  Could not fix tag %s, deleting it\n", filepath.Base(tagName))
				}
				os.Remove(tagPath)
				fixedCount++
			}
		} else {
			// If no valid commit found, delete the tag
			if verbose {
				fmt.Printf("  No valid commit found, deleting tag %s\n", filepath.Base(tagName))
			}
			tagPath := filepath.Join(repoPath, ".git", tagName)
			os.Remove(tagPath)
			fixedCount++
		}
	}

	return fixedCount, nil
}

// FixTreeObjectsWithNullSHA fixes tree objects that contain null SHA entries
func FixTreeObjectsWithNullSHA(repoPath string, verbose bool, dryRun bool) (int, error) {
	// Use the git commands approach for actual fixing
	return FixTreeCorruptionWithGitCommands(repoPath, verbose, dryRun)
}

// RunGarbageCollection runs git gc to clean up orphaned objects
func RunGarbageCollection(repoPath string, verbose bool) error {
	// First, clean up any remaining bad references that might block GC
	if verbose {
		fmt.Println("  Cleaning up any remaining bad references...")
	}
	CleanupPackedRefs(repoPath, verbose)

	// Try to prune unreachable objects
	if verbose {
		fmt.Println("  Running git prune to remove unreachable objects...")
	}
	pruneCmd := exec.Command("git", "prune", "--expire=now")
	pruneCmd.Dir = repoPath
	pruneOutput, pruneErr := pruneCmd.CombinedOutput()
	if pruneErr != nil {
		if verbose {
			fmt.Printf("  Warning: git prune failed: %v\n", pruneErr)
			if len(pruneOutput) > 0 {
				fmt.Printf("  Output: %s\n", string(pruneOutput))
			}
		}
		// Try to clean up again and retry
		CleanupPackedRefs(repoPath, verbose)
		pruneCmd = exec.Command("git", "prune", "--expire=now")
		pruneCmd.Dir = repoPath
		pruneOutput, pruneErr = pruneCmd.CombinedOutput()
		if pruneErr != nil && verbose {
			fmt.Printf("  Prune still failing after cleanup, continuing anyway...\n")
		}
	} else if verbose && len(pruneOutput) > 0 {
		fmt.Printf("  Prune output: %s\n", string(pruneOutput))
	}

	// Then run garbage collection
	if verbose {
		fmt.Println("  Running git gc to compact repository...")
	}
	cmd := exec.Command("git", "gc", "--prune=now", "--aggressive")
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			fmt.Printf("  Warning: Garbage collection failed: %v\n", err)
			if len(output) > 0 {
				fmt.Printf("  Output: %s\n", string(output))
			}
		}
		// Try one more time with basic gc
		if verbose {
			fmt.Println("  Retrying with basic gc...")
		}
		cmd = exec.Command("git", "gc", "--prune=now")
		cmd.Dir = repoPath
		output, err = cmd.CombinedOutput()
		if err != nil && verbose {
			fmt.Printf("  GC still failing: %v\n", err)
		}
		return err
	}

	if verbose && len(output) > 0 {
		fmt.Printf("  GC output: %s\n", string(output))
	}

	return nil
}

// CleanupPackedRefs removes any remaining bad references from packed-refs
func CleanupPackedRefs(repoPath string, verbose bool) {
	// Clean up packed-refs one more time
	packedRefsPath := filepath.Join(repoPath, ".git", "packed-refs")
	if content, err := os.ReadFile(packedRefsPath); err == nil {
		lines := strings.Split(string(content), "\n")
		var newLines []string
		seenRefs := make(map[string]bool)
		modified := false

		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)

			// Skip lines that start with null-like SHA (all zeros or mostly zeros)
			if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "^") {
				parts := strings.Fields(trimmedLine)
				if len(parts) >= 2 {
					sha := parts[0]
					refName := parts[1]

					// Check if SHA starts with many zeros (likely a null SHA variant)
					if strings.HasPrefix(sha, "000000000000000000000000000000000000000") {
						if verbose {
							fmt.Printf("    Removing null-like SHA: %s\n", line)
						}
						modified = true
						continue
					}

					// Check for duplicates
					if seenRefs[refName] {
						if verbose {
							fmt.Printf("    Removing duplicate: %s\n", line)
						}
						modified = true
						continue
					}
					seenRefs[refName] = true
				}
			}

			newLines = append(newLines, line)
		}

		if modified {
			newContent := strings.Join(newLines, "\n")
			os.WriteFile(packedRefsPath, []byte(newContent), 0644)
			if verbose {
				fmt.Println("    Cleaned up packed-refs")
			}
		}
	}
}

// createFixedTree creates a new tree object without null SHA entries
func createFixedTree(repo *git.Repository, tree *object.Tree, verbose bool) (bool, error) {
	nullSHA := plumbing.NewHash("0000000000000000000000000000000000000000")
	hasNullEntries := false

	// Check if tree has null entries
	for _, entry := range tree.Entries {
		if entry.Hash == nullSHA {
			hasNullEntries = true
			if verbose {
				fmt.Printf("    Found null SHA entry: %s\n", entry.Name)
			}
		}
	}

	if !hasNullEntries {
		return false, nil
	}

	// For tree corruption, we need to use git plumbing commands
	// because go-git doesn't support creating tree objects directly
	if verbose {
		fmt.Printf("    Tree has null SHA entries - will be fixed via git commands\n")
	}

	return true, nil
}

// FixTreeCorruptionWithGitCommands uses git plumbing to fix tree corruption
func FixTreeCorruptionWithGitCommands(repoPath string, verbose bool, dryRun bool) (int, error) {
	fixedCount := 0

	// Run git fsck to find corrupted trees
	cmd := exec.Command("git", "fsck", "--full")
	cmd.Dir = repoPath
	output, _ := cmd.CombinedOutput()

	lines := strings.Split(string(output), "\n")
	var corruptedTrees []string

	for _, line := range lines {
		if strings.Contains(line, "nullSha1") || strings.Contains(line, "null sha1") {
			// Extract tree hash
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "tree" && i+1 < len(parts) {
					treeHash := strings.TrimSuffix(parts[i+1], ":")
					if len(treeHash) == 40 {
						corruptedTrees = append(corruptedTrees, treeHash)
					}
					break
				}
			}
		}
	}

	if len(corruptedTrees) == 0 {
		return 0, nil
	}

	if verbose {
		if dryRun {
			fmt.Printf("  [DRY RUN] Found %d corrupted tree(s) that would be fixed\n", len(corruptedTrees))
		} else {
			fmt.Printf("  Found %d corrupted tree(s), attempting to fix...\n", len(corruptedTrees))
		}
	}

	// For each corrupted tree, create a fixed version
	for _, treeHash := range corruptedTrees {
		if verbose {
			if dryRun {
				fmt.Printf("  [DRY RUN] Would process tree %s...\n", treeHash[:8])
			} else {
				fmt.Printf("  Processing tree %s...\n", treeHash[:8])
			}
		}

		if dryRun {
			fixedCount++
			continue
		}

		// Use git cat-file to read the raw tree object (works even with corruption)
		catFileCmd := exec.Command("git", "cat-file", "-p", treeHash)
		catFileCmd.Dir = repoPath
		treeOutput, err := catFileCmd.CombinedOutput()

		// If cat-file fails, try ls-tree with --full-tree (more permissive)
		if err != nil {
			if verbose {
				fmt.Printf("    cat-file failed, trying ls-tree...\n")
			}
			lsTreeCmd := exec.Command("git", "ls-tree", "--full-tree", treeHash)
			lsTreeCmd.Dir = repoPath
			treeOutput, err = lsTreeCmd.CombinedOutput()
			if err != nil {
				if verbose {
					fmt.Printf("    Could not read tree (both methods failed): %v\n", err)
					fmt.Printf("    Attempting to create empty tree as replacement...\n")
				}
				// If we can't read the tree at all, replace it with empty tree
				newTreeHash := EmptyTreeHash
				updated, updateErr := updateCommitsWithNewTree(repoPath, treeHash, newTreeHash, verbose)
				if updateErr == nil && updated > 0 {
					fixedCount++
					if verbose {
						fmt.Printf("    Replaced corrupted tree with empty tree, updated %d commit(s)\n", updated)
					}
				}
				continue
			}
		}

		// Parse tree entries and filter out null SHA
		treeLines := strings.Split(string(treeOutput), "\n")
		var validEntries []string
		nullEntriesFound := 0

		for _, line := range treeLines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Format: "100644 blob <hash>\t<name>"
			if strings.Contains(line, "0000000000000000000000000000000000000000") {
				nullEntriesFound++
				if verbose {
					parts := strings.Split(line, "\t")
					if len(parts) > 1 {
						fmt.Printf("    Removing null SHA entry: %s\n", parts[1])
					}
				}
				continue // Skip null SHA entries
			}

			validEntries = append(validEntries, line)
		}

		if nullEntriesFound == 0 {
			if verbose {
				fmt.Printf("    No null entries found in tree (may have been fixed already)\n")
			}
			continue
		}

		var newTreeHash string

		// Create a new tree object without null SHA entries
		if len(validEntries) == 0 {
			// All entries were null, use empty tree
			newTreeHash = EmptyTreeHash
			if verbose {
				fmt.Printf("    All entries were null, using empty tree: %s\n", newTreeHash[:8])
			}
		} else {
			// Create new tree with valid entries using git mktree
			mkTreeCmd := exec.Command("git", "mktree")
			mkTreeCmd.Dir = repoPath
			mkTreeCmd.Stdin = strings.NewReader(strings.Join(validEntries, "\n") + "\n")
			newTreeOutput, err := mkTreeCmd.CombinedOutput()
			if err != nil {
				if verbose {
					fmt.Printf("    Could not create new tree: %v\n", err)
					fmt.Printf("    Output: %s\n", string(newTreeOutput))
				}
				continue
			}

			newTreeHash = strings.TrimSpace(string(newTreeOutput))
			if verbose {
				fmt.Printf("    Created new tree: %s (removed %d null entries)\n", newTreeHash[:8], nullEntriesFound)
			}
		}

		// Find and update all commits that reference this tree
		updated, err := updateCommitsWithNewTree(repoPath, treeHash, newTreeHash, verbose)
		if err != nil {
			if verbose {
				fmt.Printf("    Could not update commits: %v\n", err)
			}
			continue
		}

		if updated > 0 {
			fixedCount++
			if verbose {
				fmt.Printf("    Updated %d commit(s) to use new tree\n", updated)
			}
		} else {
			if verbose {
				fmt.Printf("    No commits reference this tree\n")
				fmt.Printf("    Warning: No commits found referencing this tree\n")
				fmt.Printf("    The tree may be dangling (not referenced by any commit)\n")
			}
			// Still count it as fixed since we created the clean tree
			fixedCount++
		}
	}

	return fixedCount, nil
}

// updateCommitsWithNewTree updates all commits that reference an old tree to use a new tree
func updateCommitsWithNewTree(repoPath, oldTreeHash, newTreeHash string, verbose bool) (int, error) {
	updatedCount := 0

	// Find commits using this tree
	var commitsToFix []string

	// Use git for-each-ref to get all valid refs first
	refsCmd := exec.Command("git", "for-each-ref", "--format=%(refname)", "refs/heads/", "refs/tags/")
	refsCmd.Dir = repoPath
	refsOutput, err := refsCmd.CombinedOutput()
	if err != nil {
		// If we can't get refs, try to find commits directly
		if verbose {
			fmt.Printf("    Could not list refs, trying alternative method\\n")
		}
	}

	// Get all valid commit hashes
	var validRefs []string
	if err == nil {
		refLines := strings.Split(string(refsOutput), "\n")
		for _, ref := range refLines {
			ref = strings.TrimSpace(ref)
			if ref != "" {
				validRefs = append(validRefs, ref)
			}
		}
	}

	// For each valid ref, walk the commit history
	for _, ref := range validRefs {
		logCmd := exec.Command("git", "log", ref, "--format=%H %T")
		logCmd.Dir = repoPath
		logOutput, err := logCmd.CombinedOutput()
		if err != nil {
			continue // Skip bad refs
		}

		lines := strings.Split(string(logOutput), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				commitHash := parts[0]
				treeHash := parts[1]

				if treeHash == oldTreeHash {
					// Check if we already have this commit
					found := false
					for _, c := range commitsToFix {
						if c == commitHash {
							found = true
							break
						}
					}
					if !found {
						commitsToFix = append(commitsToFix, commitHash)
					}
				}
			}
		}
	}

	if len(commitsToFix) == 0 {
		if verbose {
			fmt.Printf("    No commits reference this tree\n")
		}
		return 0, nil
	}

	if verbose {
		fmt.Printf("    Found %d commit(s) using this tree\n", len(commitsToFix))
	}

	// For each commit, create a replace reference with the new tree
	for _, commitHash := range commitsToFix {
		if verbose {
			fmt.Printf("    Creating replace for commit %s...\n", commitHash[:8])
		}

		// Read the commit object
		catFileCmd := exec.Command("git", "cat-file", "commit", commitHash)
		catFileCmd.Dir = repoPath
		commitData, err := catFileCmd.CombinedOutput()
		if err != nil {
			if verbose {
				fmt.Printf("    Could not read commit: %v\n", err)
			}
			continue
		}

		// Parse commit data and replace tree hash
		commitLines := strings.Split(string(commitData), "\n")
		var newCommitLines []string

		for _, line := range commitLines {
			if strings.HasPrefix(line, "tree ") {
				// Replace with new tree hash
				newCommitLines = append(newCommitLines, "tree "+newTreeHash)
			} else {
				newCommitLines = append(newCommitLines, line)
			}
		}

		newCommitData := strings.Join(newCommitLines, "\n")

		// Create new commit object using git hash-object
		hashObjCmd := exec.Command("git", "hash-object", "-t", "commit", "-w", "--stdin")
		hashObjCmd.Dir = repoPath
		hashObjCmd.Stdin = strings.NewReader(newCommitData)
		newCommitOutput, err := hashObjCmd.CombinedOutput()
		if err != nil {
			if verbose {
				fmt.Printf("    Could not create new commit: %v\n", err)
			}
			continue
		}

		newCommitHash := strings.TrimSpace(string(newCommitOutput))

		// Create replace reference
		replaceCmd := exec.Command("git", "replace", commitHash, newCommitHash)
		replaceCmd.Dir = repoPath
		err = replaceCmd.Run()
		if err != nil {
			if verbose {
				fmt.Printf("    Could not create replace ref: %v\n", err)
			}
			continue
		}

		if verbose {
			fmt.Printf("    Created replace: %s -> %s\n", commitHash[:8], newCommitHash[:8])
		}
		updatedCount++
	}

	return updatedCount, nil
}

// FixMissingCommits handles missing commit objects
func FixMissingCommits(repoPath string, verbose bool, dryRun bool) (int, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open repository: %w", err)
	}

	fixedCount := 0

	// Find all references that point to missing commits
	refs, err := repo.References()
	if err != nil {
		return 0, err
	}

	var refsToFix []string

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		// Try to get the commit
		_, err := repo.CommitObject(ref.Hash())
		if err != nil {
			// Commit is missing
			if verbose {
				if dryRun {
					fmt.Printf("  [DRY RUN] Found reference to missing commit: %s -> %s\n", ref.Name().Short(), ref.Hash().String()[:8])
				} else {
					fmt.Printf("  Found reference to missing commit: %s -> %s\n", ref.Name().Short(), ref.Hash().String()[:8])
				}
			}
			refsToFix = append(refsToFix, ref.Name().String())
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	// Fix each reference
	for _, refName := range refsToFix {
		if verbose {
			if dryRun {
				fmt.Printf("  [DRY RUN] Would fix reference: %s\n", refName)
			} else {
				fmt.Printf("  Fixing reference: %s\n", refName)
			}
		}

		if dryRun {
			fixedCount++
			continue
		}

		// Special handling for HEAD - never delete it
		if refName == "HEAD" {
			// Try to find a valid branch to point to
			validRef, findErr := findValidReference(repo)
			if findErr == nil && validRef != "" {
				headPath := filepath.Join(repoPath, ".git", "HEAD")
				newContent := fmt.Sprintf("ref: %s\n", validRef)
				if writeErr := os.WriteFile(headPath, []byte(newContent), 0644); writeErr == nil {
					if verbose {
						fmt.Printf("  Fixed HEAD -> %s\n", validRef)
					}
					fixedCount++
				}
			} else {
				// If no valid branch found, try to find any valid commit
				validCommit, findErr := findMostRecentValidCommit(repo)
				if findErr == nil && validCommit != "" {
					headPath := filepath.Join(repoPath, ".git", "HEAD")
					if writeErr := os.WriteFile(headPath, []byte(validCommit+"\n"), 0644); writeErr == nil {
						if verbose {
							fmt.Printf("  Fixed HEAD (detached) -> %s\n", validCommit[:8])
						}
						fixedCount++
					}
				}
			}
			continue
		}

		// Try to find a valid commit
		validCommit, findErr := findMostRecentValidCommit(repo)
		if findErr == nil && validCommit != "" {
			// Update the reference to point to valid commit
			refPath := filepath.Join(repoPath, ".git", refName)
			if writeErr := os.WriteFile(refPath, []byte(validCommit+"\n"), 0644); writeErr == nil {
				if verbose {
					fmt.Printf("  Fixed reference %s -> %s\n", filepath.Base(refName), validCommit[:8])
				}
				fixedCount++
			}
		} else {
			// If no valid commit found, delete the reference (but not HEAD)
			if verbose {
				fmt.Printf("  No valid commit found, deleting reference %s\n", filepath.Base(refName))
			}
			refPath := filepath.Join(repoPath, ".git", refName)
			os.Remove(refPath)
			fixedCount++
		}
	}

	return fixedCount, nil
}
