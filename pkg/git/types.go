package git

import "fmt"

// Issue represents a problem found in the repository
type Issue struct {
	Type    IssueType
	Object  string
	Message string
	Commit  string
}

type IssueType string

const (
	IssueTypeNullSHA      IssueType = "null-sha"
	IssueTypeMissingTree  IssueType = "missing-tree"
	IssueTypeMissingCommit IssueType = "missing-commit"
	IssueTypeBrokenParent IssueType = "broken-parent"
)

func (i Issue) String() string {
	return fmt.Sprintf("[%s] %s: %s", i.Type, i.Object, i.Message)
}

// BadCommit represents a commit that needs to be fixed
type BadCommit struct {
	Hash        string
	ParentHash  string // Empty if root commit
	TreeHash    string
	Author      string
	AuthorEmail string
	AuthorDate  string
	Committer   string
	CommitterEmail string
	CommitterDate string
	Message     string
	IsRoot      bool
}

func (bc BadCommit) String() string {
	if bc.IsRoot {
		return fmt.Sprintf("Commit %s (root commit)", bc.Hash[:8])
	}
	return fmt.Sprintf("Commit %s (parent: %s)", bc.Hash[:8], bc.ParentHash[:8])
}

// EmptyTreeHash is the standard Git empty tree hash
const EmptyTreeHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// EmptyBlobHash is the standard Git empty blob hash
const EmptyBlobHash = "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"

// TreeFix represents a tree that was fixed
type TreeFix struct {
	OldHash string
	NewHash string
	EntriesRemoved int
}

