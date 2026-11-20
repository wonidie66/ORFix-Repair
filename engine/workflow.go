package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// stagedResult is a complete, synchronous backend result. The Windows UI runs
// these stages in a worker goroutine and changes its buttons only on the UI thread.
type stagedResult struct {
	Phase         string
	Root          string
	Scanned       int
	Modified      int
	FaceRisks     []string
	FaceManual    []string
	Changes       []string
	Errors        []string
	RollbackItems []rollbackEntry
	LogPath       string
}

func (r stagedResult) summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "阶段：%s\r\n目录：%s\r\n扫描 INI：%d；成功修复：%d；错误：%d\r\n", r.Phase, r.Root, r.Scanned, r.Modified, len(r.Errors))
	if len(r.FaceRisks) > 0 {
		fmt.Fprintf(&b, "\r\n发现 %d 条疑似面部问题（静态风险，并非已确认故障）。\r\n", len(r.FaceRisks))
		for _, line := range r.FaceRisks {
			b.WriteString("[面部风险] " + line + "\r\n")
		}
	}
	for _, line := range r.FaceManual {
		b.WriteString("[需人工检查] " + line + "\r\n")
	}
	if len(r.Changes) > 0 {
		b.WriteString("\r\n[执行情况]\r\n")
		for _, line := range r.Changes {
			b.WriteString(line + "\r\n")
		}
	}
	if len(r.Errors) > 0 {
		b.WriteString("\r\n[错误]\r\n")
		for _, line := range r.Errors {
			b.WriteString(line + "\r\n")
		}
	}
	if r.LogPath != "" {
		b.WriteString("\r\n日志：" + r.LogPath + "\r\n")
	}
	return b.String()
}

// runNormalStage intentionally does not mutate face bindings. We rescan faces
// *after* all regular edits so the UI reports risk in the actual disk state.
func runNormalStage(root string, when time.Time) stagedResult {
	result := stagedResult{Phase: "正常增强修复", Root: root}
	r := scan(root, "enhanced")
	result.Scanned = r.scanned
	result.Errors = append(result.Errors, r.errors...)
	loglines := []string{}
	entries := []rollbackEntry{}
	for _, p := range r.repairs {
		changed, backs, err := applyGroup(p, when)
		if err != nil {
			result.Errors = append(result.Errors, rel(root, p.path)+": "+err.Error())
			loglines = append(loglines, "失败："+rel(root, p.path)+" | "+err.Error())
			continue
		}
		result.Modified++
		for i, entry := range changed {
			entry.Path = rel(root, entry.Path)
			entry.Backup = rel(root, entry.Backup)
			entries = append(entries, entry)
			loglines = append(loglines, "已修复："+entry.Path+" | 备份："+entry.Backup)
			_ = backs[i]
		}
	}
	result.RollbackItems = entries
	if len(entries) > 0 {
		name, err := writeRollback(root, when, entries)
		if err != nil {
			result.Errors = append(result.Errors, "回滚清单写入失败: "+err.Error())
		} else {
			loglines = append(loglines, "回滚清单："+rel(root, name))
		}
	}
	// Do not use the pre-repair risk report here: the user is deciding after
	// normal repair and after looking at the live model in the game.
	_, faces, manual, faceErrs := scanFaceFix(root)
	result.FaceRisks = faces
	result.FaceManual = manual
	result.Errors = append(result.Errors, faceErrs...)
	result.Changes = loglines
	r.faceFindings = faces
	output := reportText(r, "enhanced", when, false, loglines)
	if len(faceErrs) > 0 {
		output += "\n[Face Fix 扫描错误]\n" + strings.Join(faceErrs, "\n") + "\n"
	}
	file, err := outputLog(root, when, output)
	if err != nil {
		result.Errors = append(result.Errors, "日志写入失败: "+err.Error())
	} else {
		result.LogPath = file
	}
	return result
}

// mergeRollbackEntries makes the latest rollback represent the entire GUI
// session. If Face Fix modifies the same INI already modified in stage one,
// retain its *original* backup but use the latest patched checksum.
func mergeRollbackEntries(before, after []rollbackEntry) ([]rollbackEntry, error) {
	result := append([]rollbackEntry{}, before...)
	pos := map[string]int{}
	for i, e := range result {
		pos[strings.ToLower(filepath.Clean(e.Path))] = i
	}
	for _, e := range after {
		key := strings.ToLower(filepath.Clean(e.Path))
		if i, exists := pos[key]; exists {
			if !strings.EqualFold(result[i].PatchedSHA256, e.OriginalSHA256) {
				return nil, fmt.Errorf("跨阶段文件 Hash 不连续: %s", e.Path)
			}
			result[i].PatchedSHA256 = e.PatchedSHA256
		} else {
			pos[key] = len(result)
			result = append(result, e)
		}
	}
	return result, nil
}

func runFaceStage(root string, normal []rollbackEntry, when time.Time) stagedResult {
	result := stagedResult{Phase: "用户确认后的 Face Fix", Root: root}
	plans, findings, manual, errs := scanFaceFix(root)
	result.Scanned = len(plans)
	result.FaceRisks = findings
	result.FaceManual = manual
	result.Errors = append(result.Errors, errs...)
	logs := []string{}
	changedAll := []rollbackEntry{}
	for _, p := range plans {
		entries, _, err := applyGroup(p, when)
		if err != nil {
			result.Errors = append(result.Errors, rel(root, p.path)+": "+err.Error())
			logs = append(logs, "失败："+rel(root, p.path)+" | "+err.Error())
			continue
		}
		result.Modified++
		for _, entry := range entries {
			entry.Path = rel(root, entry.Path)
			entry.Backup = rel(root, entry.Backup)
			changedAll = append(changedAll, entry)
			logs = append(logs, "已完成 Face Fix："+entry.Path+" | 备份："+entry.Backup)
		}
	}
	if len(changedAll) > 0 {
		merged, err := mergeRollbackEntries(normal, changedAll)
		if err != nil {
			result.Errors = append(result.Errors, "无法合并两阶段回滚记录: "+err.Error())
			merged = changedAll
		}
		manifest, err := writeRollback(root, when, merged)
		if err != nil {
			result.Errors = append(result.Errors, "保存回滚清单失败: "+err.Error())
		} else {
			logs = append(logs, "当前会话完整回滚清单："+rel(root, manifest))
		}
		result.RollbackItems = merged
	} else {
		result.RollbackItems = normal
	}
	result.Changes = logs
	var sb strings.Builder
	fmt.Fprintf(&sb, "ORFix Repair %s | GUI 用户选择继续 Face Fix | %s\n目录：%s\n完成修改：%d 个 INI\n", version, when.Format("2006-01-02 15:04:05"), root, result.Modified)
	sb.WriteString("\n[疑似面部问题（并非静态确诊）]\n" + strings.Join(findings, "\n") + "\n")
	sb.WriteString("\n[人工检查/未修复]\n" + strings.Join(manual, "\n") + "\n")
	sb.WriteString("\n[执行与备份]\n" + strings.Join(logs, "\n") + "\n")
	sb.WriteString("\n[错误]\n" + strings.Join(result.Errors, "\n") + "\n")
	file, err := outputLog(root, when, sb.String())
	if err != nil {
		result.Errors = append(result.Errors, "日志写入失败: "+err.Error())
	} else {
		result.LogPath = file
	}
	return result
}

func previewStage(root string, when time.Time) stagedResult {
	r := scan(root, "enhanced")
	return stagedResult{Phase: "仅预览（未改动文件）", Root: root, Scanned: r.scanned, Modified: 0,
		FaceRisks: r.faceFindings, FaceManual: r.manual,
		Changes: func() []string {
			a := []string{}
			for _, p := range r.repairs {
				a = append(a, fmt.Sprintf("待修复: %s（%d项）", rel(root, p.path), len(p.changes)))
			}
			return a
		}(),
		Errors: r.errors,
		LogPath: func() string {
			name, err := outputLog(root, when, reportText(r, "enhanced", when, true, nil))
			if err != nil {
				return ""
			}
			return name
		}(),
	}
}

func rollbackStage(root string, when time.Time) stagedResult {
	n, lines := restoreLatest(root, when)
	result := stagedResult{Phase: "恢复最近一次修复", Root: root, Modified: n, Changes: lines}
	f, err := outputLog(root, when, "ORFix Repair GUI 回滚结果\n"+strings.Join(lines, "\n"))
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		result.LogPath = f
	}
	return result
}

// A helper intended for tests and users running the GUI: verify the root is
// a directory, not the bundled EXE or arbitrary working directory.
func validateRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("尚未选择 Mod 文件夹")
	}
	abs, err := filepath.Abs(strings.TrimSpace(strings.Trim(root, `"`)))
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("不是文件夹: %s", abs)
	}
	return abs, nil
}
