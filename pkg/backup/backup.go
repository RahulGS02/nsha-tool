package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BackupInfo contains information about a backup
type BackupInfo struct {
	BackupPath   string
	Timestamp    time.Time
	OriginalPath string
	Size         int64
	Method       string // "bundle" or "directory-copy"
}

// CreateBackup creates a full backup of the repository before modifications
// This includes ALL history, branches, tags, refs, and objects
func CreateBackup(repoPath, logDir string, verbose bool) (*BackupInfo, error) {
	if verbose {
		fmt.Println("  Creating complete repository backup with full history...")
	}

	// Create backup directory
	backupDir := filepath.Join(logDir, "backup")
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupPath := filepath.Join(backupDir, "repo.bundle")

	// Try git bundle first (preferred method for healthy repos)
	// --all ensures we capture everything including all branches, tags, and refs
	cmd := exec.Command("git", "bundle", "create", backupPath, "--all", "--branches", "--tags", "--remotes")
	cmd.Dir = repoPath
	_, err = cmd.CombinedOutput()

	// If git bundle fails (e.g., due to broken refs), fall back to directory copy
	if err != nil {
		if verbose {
			fmt.Println("  Git bundle failed, falling back to .git directory copy...")
		}
		return createDirectoryCopyBackup(repoPath, logDir, verbose)
	}

	if verbose {
		fmt.Println("  Backing up all references...")
	}

	// Backup all refs (branches, tags, remotes, etc.)
	refsBackupPath := filepath.Join(backupDir, "refs-backup.txt")
	refsCmd := exec.Command("git", "for-each-ref", "--format=%(refname) %(objectname) %(objecttype)")
	refsCmd.Dir = repoPath
	refsOutput, err := refsCmd.CombinedOutput()
	if err != nil {
		if verbose {
			fmt.Printf("  Warning: Could not backup refs: %v\n", err)
		}
	} else {
		err = os.WriteFile(refsBackupPath, refsOutput, 0644)
		if err != nil {
			if verbose {
				fmt.Printf("  Warning: Could not write refs backup: %v\n", err)
			}
		}
	}

	// Backup packed-refs if it exists (contains packed references)
	packedRefsPath := filepath.Join(repoPath, ".git", "packed-refs")
	if _, err := os.Stat(packedRefsPath); err == nil {
		if verbose {
			fmt.Println("  Backing up packed-refs...")
		}
		packedRefsBackup := filepath.Join(backupDir, "packed-refs")
		input, _ := os.ReadFile(packedRefsPath)
		os.WriteFile(packedRefsBackup, input, 0644)
	}

	// Backup HEAD
	headPath := filepath.Join(repoPath, ".git", "HEAD")
	if _, err := os.Stat(headPath); err == nil {
		headBackup := filepath.Join(backupDir, "HEAD")
		input, _ := os.ReadFile(headPath)
		os.WriteFile(headBackup, input, 0644)
	}

	// Backup config
	configPath := filepath.Join(repoPath, ".git", "config")
	if _, err := os.Stat(configPath); err == nil {
		configBackup := filepath.Join(backupDir, "config")
		input, _ := os.ReadFile(configPath)
		os.WriteFile(configBackup, input, 0644)
	}

	// Get backup size
	stat, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup: %w", err)
	}

	info := &BackupInfo{
		BackupPath:   backupPath,
		Timestamp:    time.Now(),
		OriginalPath: repoPath,
		Size:         stat.Size(),
		Method:       "bundle",
	}

	if verbose {
		fmt.Printf("  Backup created: %s (%.2f MB)\n", backupPath, float64(info.Size)/(1024*1024))
	}

	// Write backup info file
	infoPath := filepath.Join(backupDir, "backup-info.txt")
	infoContent := fmt.Sprintf(`Repository Backup Information
═══════════════════════════════════════════════════════════

Original Repository: %s
Backup Location: %s
Backup Time: %s
Backup Size: %.2f MB

Restore Instructions:
═══════════════════════════════════════════════════════════

To restore this backup:

1. Navigate to the repository:
   cd %s

2. Restore from bundle:
   git bundle verify %s
   git fetch %s refs/heads/*:refs/heads/*
   git fetch %s refs/tags/*:refs/tags/*

3. Or clone from bundle to a new location:
   git clone %s restored-repo

Note: This backup was created before NSHA modifications.
`, repoPath, backupPath, info.Timestamp.Format("2006-01-02 15:04:05"),
		float64(info.Size)/(1024*1024), repoPath, backupPath, backupPath, backupPath, backupPath)

	err = os.WriteFile(infoPath, []byte(infoContent), 0644)
	if err != nil {
		if verbose {
			fmt.Printf("  Warning: Could not write backup info: %v\n", err)
		}
	}

	return info, nil
}

// createDirectoryCopyBackup creates a backup by copying the entire repository folder
// This is used as a fallback when git bundle fails (e.g., due to broken refs)
func createDirectoryCopyBackup(repoPath, logDir string, verbose bool) (*BackupInfo, error) {
	if verbose {
		fmt.Println("  Copying entire repository folder to ensure complete backup...")
	}

	backupDir := filepath.Join(logDir, "backup")
	repoBackupDir := filepath.Join(backupDir, "repository")

	// Check if repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository not found at %s", repoPath)
	}

	// Copy entire repository folder recursively
	err := copyDir(repoPath, repoBackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to copy repository: %w", err)
	}

	// Calculate total size
	var totalSize int64
	filepath.Walk(repoBackupDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	info := &BackupInfo{
		BackupPath:   repoBackupDir,
		Timestamp:    time.Now(),
		OriginalPath: repoPath,
		Size:         totalSize,
		Method:       "directory-copy",
	}

	if verbose {
		fmt.Printf("  Backup created: %s (%.2f MB)\n", repoBackupDir, float64(totalSize)/(1024*1024))
	}

	// Write backup info file
	infoPath := filepath.Join(backupDir, "backup-info.txt")
	infoContent := fmt.Sprintf(`Repository Backup Information
═══════════════════════════════════════════════════════════

Original Repository: %s
Backup Location: %s
Backup Method: Complete Repository Copy
Backup Time: %s
Backup Size: %.2f MB

Restore Instructions:
═══════════════════════════════════════════════════════════

To restore this backup:

1. Stop any operations on the repository

2. Backup current repository (if needed):
   mv %s %s.old

3. Restore from backup:
   cp -r %s %s

Note: This backup contains the COMPLETE repository folder including
all files, .git directory, objects, refs, and configuration files.
This method was used because git bundle failed (likely due to broken
references).
`, repoPath, repoBackupDir, info.Timestamp.Format("2006-01-02 15:04:05"),
		float64(totalSize)/(1024*1024), repoPath, repoPath, repoBackupDir, repoPath)

	err = os.WriteFile(infoPath, []byte(infoContent), 0644)
	if err != nil {
		if verbose {
			fmt.Printf("  Warning: Could not write backup info: %v\n", err)
		}
	}

	return info, nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		// Skip .nsha or nsha directories to avoid backing up old backups
		if entry.Name() == ".nsha" || entry.Name() == "nsha" {
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Copy file
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// VerifyBackup verifies that a backup is valid
// For bundle backups, it uses git bundle verify
// For directory-copy backups, it checks if the directory exists and contains .git
func VerifyBackup(backupInfo *BackupInfo, verbose bool) error {
	if verbose {
		fmt.Printf("  Verifying backup: %s\n", backupInfo.BackupPath)
	}

	if backupInfo.Method == "bundle" {
		// Verify git bundle
		cmd := exec.Command("git", "bundle", "verify", backupInfo.BackupPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("bundle verification failed: %w\nOutput: %s", err, string(output))
		}

		if verbose {
			fmt.Println("  [SUCCESS] Bundle backup verified successfully")
		}
	} else if backupInfo.Method == "directory-copy" {
		// Verify directory copy backup
		// Check if backup directory exists
		if _, err := os.Stat(backupInfo.BackupPath); os.IsNotExist(err) {
			return fmt.Errorf("backup directory not found: %s", backupInfo.BackupPath)
		}

		// Check if .git directory exists in backup
		gitDir := filepath.Join(backupInfo.BackupPath, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			return fmt.Errorf("backup directory does not contain .git folder: %s", backupInfo.BackupPath)
		}

		if verbose {
			fmt.Println("  [SUCCESS] Directory copy backup verified successfully")
		}
	} else {
		return fmt.Errorf("unknown backup method: %s", backupInfo.Method)
	}

	return nil
}
