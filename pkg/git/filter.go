package git

import (
	"fmt"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// FilterRepo rewrites repository history to apply replace references permanently
// This is the equivalent of git filter-repo for our use case
func FilterRepo(repoPath string, force bool) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get all replace refs
	replaceMap, err := getReplaceRefs(repo)
	if err != nil {
		return fmt.Errorf("failed to get replace refs: %w", err)
	}

	if len(replaceMap) == 0 {
		return fmt.Errorf("no replace refs found - nothing to rewrite")
	}

	fmt.Printf("Found %d replace reference(s)\n", len(replaceMap))

	// Build commit mapping (old hash -> new hash)
	commitMap := make(map[plumbing.Hash]plumbing.Hash)
	
	// First, add direct replacements
	for oldHash, newHash := range replaceMap {
		commitMap[plumbing.NewHash(oldHash)] = plumbing.NewHash(newHash)
	}

	// Get all commits in topological order
	commits, err := getAllCommitsTopological(repo)
	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}

	fmt.Printf("Rewriting %d commit(s)...\n", len(commits))

	// Rewrite commits
	for _, oldHash := range commits {
		newHash, err := rewriteCommit(repo, oldHash, commitMap)
		if err != nil {
			return fmt.Errorf("failed to rewrite commit %s: %w", oldHash, err)
		}
		
		// Only add to map if it changed
		if newHash != oldHash {
			commitMap[oldHash] = newHash
		}
	}

	// Update all references
	err = updateAllReferences(repo, commitMap)
	if err != nil {
		return fmt.Errorf("failed to update references: %w", err)
	}

	return nil
}

// getReplaceRefs gets all replace references as a map
func getReplaceRefs(repo *git.Repository) (map[string]string, error) {
	refs, err := repo.References()
	if err != nil {
		return nil, err
	}

	replaceMap := make(map[string]string)

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		refName := ref.Name().String()
		if len(refName) > 13 && refName[:13] == "refs/replace/" {
			oldHash := refName[13:]
			newHash := ref.Hash().String()
			replaceMap[oldHash] = newHash
			fmt.Printf("  Replace: %s -> %s\n", oldHash[:8], newHash[:8])
		}
		return nil
	})

	return replaceMap, err
}

// getAllCommitsTopological returns all commits in topological order (parents before children)
func getAllCommitsTopological(repo *git.Repository) ([]plumbing.Hash, error) {
	// Get all references
	refs, err := repo.References()
	if err != nil {
		return nil, err
	}

	// Collect all commit hashes
	commitSet := make(map[plumbing.Hash]bool)
	var startCommits []plumbing.Hash

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		// Skip replace refs
		if len(ref.Name().String()) > 13 && ref.Name().String()[:13] == "refs/replace/" {
			return nil
		}

		if ref.Name().IsBranch() || ref.Name().IsTag() {
			startCommits = append(startCommits, ref.Hash())
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Walk all commits from all refs
	for _, startHash := range startCommits {
		err := walkCommits(repo, startHash, commitSet)
		if err != nil {
			// Continue even if some commits are broken
			continue
		}
	}

	// Convert to slice and sort (simple approach - by hash string for determinism)
	var commits []plumbing.Hash
	for hash := range commitSet {
		commits = append(commits, hash)
	}

	// Sort to ensure parents are processed before children
	sort.Slice(commits, func(i, j int) bool {
		// Try to ensure parents come first
		ci, _ := repo.CommitObject(commits[i])
		cj, _ := repo.CommitObject(commits[j])
		
		if ci != nil && cj != nil {
			// If i is parent of j, i should come first
			for _, parent := range cj.ParentHashes {
				if parent == commits[i] {
					return true
				}
			}
			// If j is parent of i, j should come first
			for _, parent := range ci.ParentHashes {
				if parent == commits[j] {
					return false
				}
			}
		}
		
		return commits[i].String() < commits[j].String()
	})

	return commits, nil
}

// walkCommits recursively walks commit history
func walkCommits(repo *git.Repository, hash plumbing.Hash, visited map[plumbing.Hash]bool) error {
	if visited[hash] {
		return nil
	}

	visited[hash] = true

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return err
	}

	// Walk parents
	for _, parentHash := range commit.ParentHashes {
		walkCommits(repo, parentHash, visited)
	}

	return nil
}

// rewriteCommit rewrites a single commit, updating its parents based on the commit map
func rewriteCommit(repo *git.Repository, oldHash plumbing.Hash, commitMap map[plumbing.Hash]plumbing.Hash) (plumbing.Hash, error) {
	// If this commit is directly replaced, return the replacement
	if newHash, exists := commitMap[oldHash]; exists {
		return newHash, nil
	}

	// Get the original commit
	oldCommit, err := repo.CommitObject(oldHash)
	if err != nil {
		// If we can't read it, it might be broken - check if it's in replace map
		return oldHash, nil
	}

	// Check if any parents need to be rewritten
	needsRewrite := false
	newParents := make([]plumbing.Hash, 0, len(oldCommit.ParentHashes))

	for _, parentHash := range oldCommit.ParentHashes {
		if newParentHash, exists := commitMap[parentHash]; exists {
			newParents = append(newParents, newParentHash)
			needsRewrite = true
		} else {
			newParents = append(newParents, parentHash)
		}
	}

	// If no rewrite needed, return original hash
	if !needsRewrite {
		return oldHash, nil
	}

	// Create new commit with updated parents
	newCommit := &object.Commit{
		Author:       oldCommit.Author,
		Committer:    oldCommit.Committer,
		Message:      oldCommit.Message,
		TreeHash:     oldCommit.TreeHash,
		ParentHashes: newParents,
	}

	// Store the new commit
	obj := repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.CommitObject)

	err = newCommit.Encode(obj)
	if err != nil {
		return oldHash, fmt.Errorf("failed to encode commit: %w", err)
	}

	newHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return oldHash, fmt.Errorf("failed to store commit: %w", err)
	}

	return newHash, nil
}

// updateAllReferences updates all branch and tag references to point to rewritten commits
func updateAllReferences(repo *git.Repository, commitMap map[plumbing.Hash]plumbing.Hash) error {
	refs, err := repo.References()
	if err != nil {
		return err
	}

	var refsToUpdate []struct {
		name    plumbing.ReferenceName
		oldHash plumbing.Hash
		newHash plumbing.Hash
	}

	// Collect refs that need updating
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		// Skip replace refs
		refName := ref.Name().String()
		if len(refName) > 13 && refName[:13] == "refs/replace/" {
			return nil
		}

		// Only update branches and tags
		if !ref.Name().IsBranch() && !ref.Name().IsTag() {
			return nil
		}

		oldHash := ref.Hash()

		// Check if this ref points to a rewritten commit
		if newHash, exists := commitMap[oldHash]; exists {
			refsToUpdate = append(refsToUpdate, struct {
				name    plumbing.ReferenceName
				oldHash plumbing.Hash
				newHash plumbing.Hash
			}{ref.Name(), oldHash, newHash})
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Update refs
	for _, update := range refsToUpdate {
		newRef := plumbing.NewHashReference(update.name, update.newHash)
		err = repo.Storer.SetReference(newRef)
		if err != nil {
			return fmt.Errorf("failed to update %s: %w", update.name, err)
		}
		fmt.Printf("  Updated %s: %s -> %s\n", update.name.Short(), update.oldHash.String()[:8], update.newHash.String()[:8])
	}

	return nil
}

// GetReplaceRefs returns all replace references (exported for use in commands)
func GetReplaceRefs(repoPath string) (map[string]string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}
	return getReplaceRefs(repo)
}

