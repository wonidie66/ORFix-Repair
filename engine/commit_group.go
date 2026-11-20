package engine

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// applyGroup ensures that both a historical Blend remap and its INI update
// succeed together. Pre-flight checks all files and hashes; on a later write
// failure, previously committed files are automatically restored, with their
// timestamped backups retained for inspection.
func applyGroup(p plan, nowStamp time.Time) ([]rollbackEntry, []string, error) {
	edits := append([]plan{}, p.companions...)
	edits = append(edits, p)
	seen := map[string]bool{}
	for _, item := range edits {
		if seen[item.path] {
			return nil, nil, errors.New("同一文件被重复列入修复计划")
		}
		seen[item.path] = true
		b, e := os.ReadFile(item.path)
		if e != nil {
			return nil, nil, fmt.Errorf("预检查读取失败 %s: %w", item.path, e)
		}
		if sha256.Sum256(b) != item.digest {
			return nil, nil, fmt.Errorf("预检查 Hash 不一致 %s", item.path)
		}
		st, e := os.Stat(item.path)
		if e != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0200 == 0 {
			return nil, nil, fmt.Errorf("目标不是可写普通文件 %s", item.path)
		}
	}
	committed := []plan{}
	manifest := []rollbackEntry{}
	backups := []string{}
	for _, item := range edits {
		backup, err := backupAndReplace(item, nowStamp)
		if err != nil {
			errorsDuringUndo := []string{}
			for i := len(committed) - 1; i >= 0; i-- {
				previous := committed[i]
				current, _ := os.ReadFile(previous.path)
				if sha256.Sum256(current) != sha256.Sum256(previous.patched) {
					errorsDuringUndo = append(errorsDuringUndo, "回退前 Hash 不匹配: "+previous.path)
					continue
				}
				if e := atomicRestore(previous.path, previous.original); e != nil {
					errorsDuringUndo = append(errorsDuringUndo, previous.path+": "+e.Error())
				}
			}
			if len(errorsDuringUndo) > 0 {
				return nil, backups, fmt.Errorf("%w；自动回退部分失败：%v", err, errorsDuringUndo)
			}
			return nil, backups, fmt.Errorf("%w；已将此前成功写入的文件自动回退", err)
		}
		committed = append(committed, item)
		backups = append(backups, backup)
		manifest = append(manifest, rollbackEntry{Path: item.path, Backup: backup, OriginalSHA256: hexSha(item.original), PatchedSHA256: hexSha(item.patched)})
	}
	return manifest, backups, nil
}

func atomicRestore(path string, data []byte) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".orfix-undo-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Chmod(st.Mode().Perm()); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
