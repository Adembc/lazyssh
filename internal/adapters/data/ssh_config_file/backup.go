// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ssh_config_file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// createBackup creates a timestamped backup of the current config file
func (r *Repository) createBackup() error {
	if _, err := r.fileSystem.Stat(r.configPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check if config file exists: %w", err)
	}

	timestamp := time.Now().UnixMilli()
	backupPath := fmt.Sprintf("%s-%d-%s", r.configPath, timestamp, BackupSuffix)

	if err := r.copyFile(r.configPath, backupPath); err != nil {
		return fmt.Errorf("failed to copy config to backup: %w", err)
	}

	r.logger.Infof("Created backup: %s", backupPath)

	configDir := filepath.Dir(r.configPath)

	backupFiles, err := r.findBackupFiles(configDir)
	if err != nil {
		return err
	}

	if len(backupFiles) <= MaxBackups {
		return nil
	}

	sort.Slice(backupFiles, func(i, j int) bool {
		return backupFiles[i].ModTime().After(backupFiles[j].ModTime())
	})

	for i := MaxBackups; i < len(backupFiles); i++ {
		backupPath := filepath.Join(configDir, backupFiles[i].Name())
		if err := r.fileSystem.Remove(backupPath); err != nil {
			r.logger.Warnf("failed to remove old backup %s: %v", backupPath, err)
			continue
		}
		r.logger.Infof("Removed old backup: %s", backupPath)
	}
	return nil
}

// copyFile copies a file from src to dst
func (r *Repository) copyFile(src, dst string) error {
	srcFile, err := r.fileSystem.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := srcFile.Close(); cerr != nil {
			r.logger.Warnf("failed to close source file %s: %v", src, cerr)
		}
	}()

	srcInfo, err := r.fileSystem.Stat(src)
	if err != nil {
		return err
	}

	destFile, err := r.fileSystem.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if cerr := destFile.Close(); cerr != nil {
			r.logger.Warnf("failed to close destination file %s: %v", dst, cerr)
		}
	}()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// findBackupFiles finds all backup files for the given config file
func (r *Repository) findBackupFiles(dir string) ([]os.FileInfo, error) {
	entries, err := r.fileSystem.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var backupFiles []os.FileInfo

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, BackupSuffix) {
			info, err := entry.Info()
			if err != nil {
				r.logger.Warnf("failed to get info for backup file %s: %v", name, err)
				continue
			}
			backupFiles = append(backupFiles, info)
		}
	}

	return backupFiles, nil
}
