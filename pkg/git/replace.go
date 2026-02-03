package git

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// CreateEmptyTree creates an empty tree object in the repository
func CreateEmptyTree(repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Create empty tree
	tree := &object.Tree{
		Entries: []object.TreeEntry{},
	}

	obj := repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.TreeObject)
	
	err = tree.Encode(obj)
	if err != nil {
		return "", fmt.Errorf("failed to encode tree: %w", err)
	}

	hash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", fmt.Errorf("failed to store tree: %w", err)
	}

	return hash.String(), nil
}

// ReplaceCommit creates a replace reference for a bad commit
func ReplaceCommit(repoPath string, badCommit BadCommit) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get the bad commit
	hash := plumbing.NewHash(badCommit.Hash)
	oldCommit, err := repo.CommitObject(hash)
	if err != nil {
		// If we can't read the commit, create a minimal one
		return createMinimalReplacement(repo, badCommit)
	}

	// Get or create empty tree
	emptyTreeHash := plumbing.NewHash(EmptyTreeHash)
	
	// Try to get the tree, if it doesn't exist, create it
	_, err = repo.TreeObject(emptyTreeHash)
	if err != nil {
		emptyTreeStr, err := CreateEmptyTree(repoPath)
		if err != nil {
			return fmt.Errorf("failed to create empty tree: %w", err)
		}
		emptyTreeHash = plumbing.NewHash(emptyTreeStr)
	}

	// Create new commit with valid tree
	newCommit := &object.Commit{
		Author:    oldCommit.Author,
		Committer: oldCommit.Committer,
		Message:   oldCommit.Message,
		TreeHash:  emptyTreeHash,
	}

	// Set parent if exists and is valid
	if badCommit.ParentHash != "" && badCommit.ParentHash != "0000000000000000000000000000000000000000" {
		parentHash := plumbing.NewHash(badCommit.ParentHash)
		// Verify parent exists
		_, err := repo.CommitObject(parentHash)
		if err == nil {
			newCommit.ParentHashes = []plumbing.Hash{parentHash}
		}
	}

	// Store the new commit
	obj := repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.CommitObject)
	
	err = newCommit.Encode(obj)
	if err != nil {
		return fmt.Errorf("failed to encode commit: %w", err)
	}

	newHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return fmt.Errorf("failed to store commit: %w", err)
	}

	// Create replace reference
	refName := plumbing.ReferenceName(fmt.Sprintf("refs/replace/%s", badCommit.Hash))
	ref := plumbing.NewHashReference(refName, newHash)

	err = repo.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("failed to create replace reference: %w", err)
	}

	return nil
}

// createMinimalReplacement creates a minimal commit when the original is unreadable
func createMinimalReplacement(repo *git.Repository, badCommit BadCommit) error {
	emptyTreeHash := plumbing.NewHash(EmptyTreeHash)
	
	// Create empty tree if it doesn't exist
	_, err := repo.TreeObject(emptyTreeHash)
	if err != nil {
		tree := &object.Tree{Entries: []object.TreeEntry{}}
		obj := repo.Storer.NewEncodedObject()
		obj.SetType(plumbing.TreeObject)
		tree.Encode(obj)
		emptyTreeHash, _ = repo.Storer.SetEncodedObject(obj)
	}

	now := time.Now()
	sig := object.Signature{
		Name:  "NSHA Tool",
		Email: "nsha@fix.local",
		When:  now,
	}

	newCommit := &object.Commit{
		Author:    sig,
		Committer: sig,
		Message:   badCommit.Message,
		TreeHash:  emptyTreeHash,
	}

	if badCommit.ParentHash != "" {
		newCommit.ParentHashes = []plumbing.Hash{plumbing.NewHash(badCommit.ParentHash)}
	}

	obj := repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.CommitObject)
	newCommit.Encode(obj)
	
	newHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return err
	}

	refName := plumbing.ReferenceName(fmt.Sprintf("refs/replace/%s", badCommit.Hash))
	ref := plumbing.NewHashReference(refName, newHash)
	
	return repo.Storer.SetReference(ref)
}

// CleanupReplaceRefs removes all replace references
func CleanupReplaceRefs(repoPath string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("failed to get references: %w", err)
	}

	var replaceRefs []string
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().String()[:13] == "refs/replace/" {
			replaceRefs = append(replaceRefs, ref.Name().String())
		}
		return nil
	})

	if err != nil {
		return err
	}

	for _, refName := range replaceRefs {
		err = repo.Storer.RemoveReference(plumbing.ReferenceName(refName))
		if err != nil {
			return fmt.Errorf("failed to remove %s: %w", refName, err)
		}
	}

	return nil
}

