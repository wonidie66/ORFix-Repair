package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const rollbackName = "ORFixRepair_latest_rollback.json"

type rollbackEntry struct {
	Path           string `json:"ini"`
	Backup         string `json:"backup"`
	OriginalSHA256 string `json:"original_sha256"`
	PatchedSHA256  string `json:"patched_sha256"`
}
type rollbackManifest struct {
	Format  int             `json:"format"`
	Created string          `json:"created"`
	Items   []rollbackEntry `json:"items"`
}

func hexSha(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
func safeWithin(root, rel string) (string, error) {
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("不安全的相对路径: %q", rel)
	}
	abs := filepath.Join(root, filepath.Clean(rel))
	v, err := filepath.Rel(root, abs)
	if err != nil || !filepath.IsLocal(v) {
		return "", errors.New("路径超出扫描根目录")
	}
	return abs, nil
}
func writeRollback(root string, t time.Time, items []rollbackEntry) (string, error) {
	if len(items) == 0 {
		return "", nil
	}
	m := rollbackManifest{Format: 1, Created: t.Format(time.RFC3339), Items: items}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	path := filepath.Join(root, "ORFixRepair_"+t.Format("2006-01-02_15-04-05")+"_rollback.json")
	for i := 0; i < 999; i++ {
		candidate := path
		if i > 0 {
			candidate = strings.TrimSuffix(path, ".json") + fmt.Sprintf("_%d.json", i+1)
		}
		f, e := os.OpenFile(candidate, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if os.IsExist(e) {
			continue
		}
		if e != nil {
			return "", e
		}
		_, e = f.Write(data)
		if e == nil {
			e = f.Sync()
		}
		ce := f.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			return "", e
		}
		// latest pointer copied atomically to provide a deterministic 1-click rollback.
		temp, e := os.CreateTemp(root, ".orfix-rollback-*.tmp")
		if e != nil {
			return candidate, e
		}
		_, e = temp.Write(data)
		if e == nil {
			e = temp.Sync()
		}
		ce = temp.Close()
		if e == nil {
			e = ce
		}
		if e == nil {
			e = os.Rename(temp.Name(), filepath.Join(root, rollbackName))
		}
		os.Remove(temp.Name())
		return candidate, e
	}
	return "", errors.New("回滚清单文件名冲突过多")
}
func restoreLatest(root string, now time.Time) (int, []string) {
	b, err := os.ReadFile(filepath.Join(root, rollbackName))
	if err != nil {
		return 0, []string{"读取最近一次修复记录失败：" + err.Error()}
	}
	var m rollbackManifest
	if err := json.Unmarshal(b, &m); err != nil || m.Format != 1 {
		return 0, []string{"最近一次修复记录格式无效"}
	}
	restored := 0
	var logs []string
	for _, e := range m.Items {
		path, pErr := safeWithin(root, e.Path)
		backup, bErr := safeWithin(root, e.Backup)
		if pErr != nil || bErr != nil {
			logs = append(logs, "跳过不安全路径："+e.Path)
			continue
		}
		curr, err := os.ReadFile(path)
		if err != nil {
			logs = append(logs, "读原文件失败："+e.Path+" "+err.Error())
			continue
		}
		if hexSha(curr) != e.PatchedSHA256 {
			logs = append(logs, "跳过：文件修复后又被其他操作修改："+e.Path)
			continue
		}
		before, err := os.ReadFile(backup)
		if err != nil {
			logs = append(logs, "备份丢失："+e.Backup)
			continue
		}
		if hexSha(before) != e.OriginalSHA256 {
			logs = append(logs, "备份哈希校验失败："+e.Backup)
			continue
		}
		// Keep backup of the patched state as well; no data is silently discarded.
		p := plan{path: path, original: curr, digest: sha256.Sum256(curr), patched: before}
		newBackup, err := backupAndReplace(p, now)
		if err != nil {
			logs = append(logs, "回滚失败："+e.Path+" "+err.Error())
			continue
		}
		restored++
		logs = append(logs, "已恢复："+e.Path+" | 回滚前状态备份："+rel(root, newBackup))
	}
	if restored > 0 { // Prevent accidental second rollback to a different state.
		os.Rename(filepath.Join(root, rollbackName), filepath.Join(root, "ORFixRepair_"+now.Format("2006-01-02_15-04-05")+"_已回滚.json"))
	}
	return restored, logs
}
