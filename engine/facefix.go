package engine

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Face Fix is intentionally opt-in. Static INI analysis can identify a
// potentially incompatible face diffuse binding, but cannot prove that the
// user will see white circles or missing makeup in-game.
var (
	facePS0  = regexp.MustCompile(`(?i)^([ \t]*)ps-t0([ \t]*=[ \t]*)(Resource[A-Za-z0-9_.-]+)([ \t]*(?:;.*)?)$`)
	faceThis = regexp.MustCompile(`(?i)^\s*this\s*=\s*(Resource[A-Za-z0-9_.-]+)\s*(?:;.*)?$`)
	faceRun  = regexp.MustCompile(`(?i)^\s*run\s*=\s*(CommandList[A-Za-z0-9_.-]*(?:\\[A-Za-z0-9_.-]+)*)(?:\s*;.*)?$`)
)

type faceAnalysis struct {
	findings []string
	changes  []string
	manual   []string
	patched  []byte
}

func isFaceDiffuseOverride(name string) bool {
	n := strings.ToLower(name)
	if !strings.HasPrefix(n, "textureoverride") {
		return false
	}
	tail := name[len("TextureOverride"):]
	tokens := map[string]bool{}
	for _, tok := range reNameToken.FindAllString(tail, -1) {
		tokens[strings.ToLower(tok)] = true
	}
	if tokens["face"] && tokens["diffuse"] {
		return true
	}
	// Conservative support for legacy all-lowercase section names.
	return strings.Contains(strings.ToLower(tail), "face") && strings.Contains(strings.ToLower(tail), "diffuse")
}

func isFaceDiffuseResource(ref string) bool {
	r := strings.ToLower(ref)
	if idx := strings.LastIndex(r, "."); idx >= 0 {
		tail := r[idx+1:]
		if tail != "" && strings.IndexFunc(tail, func(c rune) bool { return c < '0' || c > '9' }) < 0 {
			r = r[:idx]
		}
	}
	return strings.HasPrefix(r, "resource") && strings.Contains(r, "face") && strings.HasSuffix(r, "diffuse")
}

func isTextureFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".dds", ".tga", ".png", ".jpg", ".jpeg", ".bmp":
		return true
	default:
		return false
	}
}

func linePS0(lines []string, sec section) []int {
	var out []int
	for i := sec.start + 1; i < sec.end; i++ {
		if facePS0.MatchString(normLine(lines[i])) {
			out = append(out, i)
		}
	}
	return out
}

func lineHasThis(lines []string, sec section) bool {
	for i := sec.start + 1; i < sec.end; i++ {
		if faceThis.MatchString(normLine(lines[i])) {
			return true
		}
	}
	return false
}

func runTargets(lines []string, sec section) []string {
	var out []string
	for i := sec.start + 1; i < sec.end; i++ {
		if m := faceRun.FindStringSubmatch(normLine(lines[i])); m != nil {
			out = append(out, strings.ToLower(m[1]))
		}
	}
	return out
}

func faceCommandGraph(ss []section, lines []string) (map[string]section, map[string][]string, map[string][]string) {
	byName := make(map[string]section, len(ss))
	graph := make(map[string][]string)
	callers := make(map[string][]string)
	for _, sec := range ss {
		byName[strings.ToLower(sec.name)] = sec
	}
	for _, sec := range ss {
		source := strings.ToLower(sec.name)
		for _, target := range runTargets(lines, sec) {
			graph[source] = append(graph[source], target)
			callers[target] = append(callers[target], source)
		}
	}
	return byName, graph, callers
}

func faceReachable(root string, graph map[string][]string, byName map[string]section) (map[string]bool, []string) {
	reachable := map[string]bool{}
	missing := []string{}
	queue := append([]string{}, graph[root]...)
	for len(queue) > 0 {
		target := queue[0]
		queue = queue[1:]
		if reachable[target] {
			continue
		}
		sec, ok := byName[target]
		if !ok {
			// External global libraries are normal. Only unresolved face-named
			// command lists are reported as ambiguous.
			globalKnown := strings.HasPrefix(target, `commandlist\global\orfix\`) ||
				strings.HasPrefix(target, `commandlist\texfx\`) ||
				strings.HasPrefix(target, `commandlist\global\texfx\`) ||
				strings.HasPrefix(target, `commandlist\global\gimi\`)
			if !globalKnown {
				missing = append(missing, target)
			}
			continue
		}
		if !strings.HasPrefix(strings.ToLower(sec.name), "commandlist") {
			continue
		}
		reachable[target] = true
		queue = append(queue, graph[target]...)
	}
	return reachable, missing
}

func applyFaceLine(line string) (string, bool) {
	cr := strings.HasSuffix(line, "\r")
	v := strings.TrimSuffix(line, "\r")
	m := facePS0.FindStringSubmatch(v)
	if m == nil {
		return line, false
	}
	out := m[1] + "this" + m[2] + m[3] + m[4]
	if cr {
		out += "\r"
	}
	return out, true
}

func analyzeFaceINI(path string, content []byte) faceAnalysis {
	result := faceAnalysis{}
	if len(content) == 0 || bytes.IndexByte(content, 0) >= 0 || strings.EqualFold(filepath.Base(path), "ORFix.ini") || isLibraryNamespace(content) {
		return result
	}
	lines := strings.Split(string(content), "\n")
	sections := parseSections(lines)
	resources := resourceFiles(lines, sections)
	byName, graph, callers := faceCommandGraph(sections, lines)
	var roots []section
	for _, sec := range sections {
		if isFaceDiffuseOverride(sec.name) {
			roots = append(roots, sec)
		}
	}
	if len(roots) == 0 {
		return result
	}

	// Include every face root in the allow-list. A CommandList shared only by
	// multiple face roots can be considered face-specific; any non-face caller
	// makes it ambiguous and therefore non-patchable.
	faceRoots := map[string]bool{}
	for _, root := range roots {
		faceRoots[strings.ToLower(root.name)] = true
	}
	allReachable := map[string]bool{}
	rootReach := map[string]map[string]bool{}
	rootMissing := map[string][]string{}
	for _, root := range roots {
		key := strings.ToLower(root.name)
		reach, missing := faceReachable(key, graph, byName)
		rootReach[key] = reach
		rootMissing[key] = missing
		for n := range reach {
			allReachable[n] = true
		}
	}

	// Detect non-face overrides that reach the very same nested commands.
	// Checking only the immediate caller misses the case Body -> SharedEntry
	// -> SharedDiffuse where Face also reaches SharedDiffuse.
	nonFaceReachable := map[string]bool{}
	for _, sec := range sections {
		name := strings.ToLower(sec.name)
		if !strings.HasPrefix(name, "textureoverride") || faceRoots[name] {
			continue
		}
		reach, _ := faceReachable(name, graph, byName)
		for command := range reach {
			nonFaceReachable[command] = true
		}
	}
	replacements := map[int]string{}
	for _, root := range roots {
		rootKey := strings.ToLower(root.name)
		var candidateLines []int
		var reasons []string
		rootT0 := linePS0(lines, root)
		if len(rootT0) > 0 && lineHasThis(lines, root) {
			reasons = append(reasons, "同一 Face 区块同时存在 ps-t0 与 this，可能存在分支差异")
		}
		for _, ln := range rootT0 {
			m := facePS0.FindStringSubmatch(normLine(lines[ln]))
			if m == nil || !isFaceDiffuseResource(m[3]) {
				reasons = append(reasons, fmt.Sprintf("第 %d 行不是可确认的面部 Diffuse 资源绑定", ln+1))
				continue
			}
			if !fileForRef(path, m[3], resources) {
				reasons = append(reasons, fmt.Sprintf("第 %d 行引用的面部贴图资源不存在或文件缺失：%s", ln+1, m[3]))
				continue
			}
			file := resources[strings.ToLower(m[3])]
			if !isTextureFile(file) {
				reasons = append(reasons, fmt.Sprintf("第 %d 行资源不是可识别的贴图文件：%s", ln+1, file))
				continue
			}
			candidateLines = append(candidateLines, ln)
		}

		// Follow Face -> CommandList -> nested CommandList. Only rewrite a
		// CommandList when it is exclusively reached from face-related roots.
		for cmd := range rootReach[rootKey] {
			sec := byName[cmd]
			cmdT0 := linePS0(lines, sec)
			if len(cmdT0) == 0 {
				if len(rootT0) > 0 && lineHasThis(lines, sec) {
					reasons = append(reasons, "面部区块直接绑定 ps-t0，但其调用链另有 this 赋值，需人工检查")
				}
				continue
			}
			shared := nonFaceReachable[cmd]
			for _, caller := range callers[cmd] {
				if !faceRoots[caller] && !allReachable[caller] {
					shared = true
					break
				}
			}
			if shared {
				reasons = append(reasons, "CommandList "+sec.name+" 同时被非面部区块调用，不能安全地全局修改")
				continue
			}
			if lineHasThis(lines, sec) {
				reasons = append(reasons, "CommandList "+sec.name+" 同时存在 ps-t0 与 this，需人工检查分支")
				continue
			}
			for _, ln := range cmdT0 {
				m := facePS0.FindStringSubmatch(normLine(lines[ln]))
				if m == nil || !isFaceDiffuseResource(m[3]) {
					reasons = append(reasons, "CommandList "+sec.name+" 含有非面部 Diffuse 的 ps-t0，不能自动修改")
					continue
				}
				if !fileForRef(path, m[3], resources) {
					reasons = append(reasons, "CommandList "+sec.name+" 引用的面部贴图文件缺失："+m[3])
					continue
				}
				file := resources[strings.ToLower(m[3])]
				if !isTextureFile(file) {
					reasons = append(reasons, "CommandList "+sec.name+" 的资源不是可识别贴图文件："+file)
					continue
				}
				candidateLines = append(candidateLines, ln)
			}
		}
		if len(rootMissing[rootKey]) > 0 {
			reasons = append(reasons, "无法解析面部调用链："+strings.Join(rootMissing[rootKey], ", "))
		}
		if len(candidateLines) == 0 && len(reasons) == 0 {
			continue
		}
		if len(candidateLines) == 0 {
			result.findings = append(result.findings, root.name+"：疑似面部兼容风险；"+strings.Join(reasons, "；"))
			result.manual = append(result.manual, root.name+"：没有可安全自动修改的 ps-t0 行；"+strings.Join(reasons, "；"))
			continue
		}
		if len(reasons) > 0 {
			result.findings = append(result.findings, root.name+"：疑似面部兼容风险，但调用链存在歧义；"+strings.Join(reasons, "；"))
			result.manual = append(result.manual, root.name+"：保留原配置，需人工检查；"+strings.Join(reasons, "；"))
			continue
		}
		for _, ln := range candidateLines {
			if _, exists := replacements[ln]; exists {
				continue
			}
			newLine, ok := applyFaceLine(lines[ln])
			if !ok {
				result.manual = append(result.manual, root.name+"：第 "+fmt.Sprint(ln+1)+" 行转换失败")
				continue
			}
			replacements[ln] = newLine
		}
		if len(candidateLines) > 0 {
			result.findings = append(result.findings, root.name+"：发现面部 Diffuse 使用 ps-t0 的疑似风险；可选 Face Fix 候选 "+fmt.Sprintf("%d", len(candidateLines))+" 行。此项不是游戏故障确认")
			result.changes = append(result.changes, root.name+"：将面部专属 Diffuse 绑定从 ps-t0 改为 this")
		}
	}

	if len(replacements) > 0 {
		for line, value := range replacements {
			lines[line] = value
		}
		result.patched = []byte(strings.Join(lines, "\n"))
	}
	return result
}

func scanFaceFix(root string) ([]plan, []string, []string, []string) {
	var plans []plan
	var findings, manual, errs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			errs = append(errs, path+"："+walkErr.Error())
			return nil
		}
		if d.IsDir() || d.Type()&os.ModeSymlink != 0 || strings.ToLower(filepath.Ext(d.Name())) != ".ini" {
			return nil
		}
		if strings.HasPrefix(strings.ToLower(d.Name()), "disabled") || strings.EqualFold(d.Name(), "ORFix.ini") {
			return nil
		}
		st, e := d.Info()
		if e != nil {
			errs = append(errs, path+"："+e.Error())
			return nil
		}
		if st.Size() > maxINIBytes {
			manual = append(manual, rel(root, path)+"：INI 文件过大，跳过")
			return nil
		}
		content, e := os.ReadFile(path)
		if e != nil {
			errs = append(errs, path+"："+e.Error())
			return nil
		}
		a := analyzeFaceINI(path, content)
		for _, f := range a.findings {
			findings = append(findings, rel(root, path)+" | "+f)
		}
		for _, m := range a.manual {
			manual = append(manual, rel(root, path)+" | "+m)
		}
		if len(a.changes) > 0 && len(a.patched) > 0 && !bytes.Equal(a.patched, content) {
			plans = append(plans, plan{path: path, original: content, digest: sha256.Sum256(content), patched: a.patched, changes: a.changes, manual: a.manual})
		}
		return nil
	})
	if err != nil {
		errs = append(errs, err.Error())
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].path < plans[j].path })
	sort.Strings(findings)
	sort.Strings(manual)
	return plans, findings, manual, errs
}

func runFaceFix(root string, now time.Time, preview, yes bool) {
	plans, findings, manual, errs := scanFaceFix(root)
	var results []string
	modified := 0
	if len(plans) > 0 && !preview {
		var detail strings.Builder
		for _, p := range plans {
			fmt.Fprintf(&detail, "%s：%d 处候选修改\n", rel(root, p.path), len(p.changes))
		}
		msg := fmt.Sprintf("发现 %d 条疑似面部风险；%d 个 INI 存在可安全定位的 Face Diffuse ps-t0 绑定。\n\n本功能只把已定位的面部 Diffuse ps-t0 改为 this，不修改 DDS、IB、BUF、其他材质槽位或共享/歧义 CommandList。\n\n%s\n确认已在游戏中观察到白色圆圈、妆容消失或面部贴图异常后再执行。是否继续？", len(findings), len(plans), detail.String())
		if yes || confirm("ORFix Repair — 可选 Face Fix", msg) {
			items := []rollbackEntry{}
			for _, p := range plans {
				entries, backups, err := applyGroup(p, now)
				if err != nil {
					results = append(results, "失败："+rel(root, p.path)+" | "+err.Error())
					continue
				}
				modified++
				for i, entry := range entries {
					entry.Path = rel(root, entry.Path)
					entry.Backup = rel(root, entry.Backup)
					items = append(items, entry)
					if i < len(backups) {
						results = append(results, "成功："+entry.Path+" | 备份："+rel(root, backups[i]))
					}
				}
			}
			if len(items) > 0 {
				manifest, err := writeRollback(root, now, items)
				if err != nil {
					results = append(results, "警告：回滚清单保存失败："+err.Error())
				} else {
					results = append(results, "一键回滚清单："+rel(root, manifest))
				}
			}
		} else {
			preview = true
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "ORFix Repair %s | 可选 Face Fix | %s\n扫描目录：%s\n候选 INI：%d；可修改：%d；人工检查：%d；错误：%d\n", version, now.Format("2006-01-02 15:04:05"), root, len(findings), len(plans), len(manual), len(errs))
	if preview {
		sb.WriteString("预览/取消模式：没有修改文件。\n")
	}
	sb.WriteString("\n[疑似面部兼容性风险（不是故障确认）]\n")
	for _, x := range findings {
		sb.WriteString(x + "\n")
	}
	if len(findings) == 0 {
		sb.WriteString("未发现当前规则可识别的 FaceHeadDiffuse ps-t0 风险。\n")
	}
	sb.WriteString("\n[人工检查 / 未自动修改]\n")
	for _, x := range manual {
		sb.WriteString(x + "\n")
	}
	if len(manual) == 0 {
		sb.WriteString("无\n")
	}
	sb.WriteString("\n[执行与备份]\n")
	for _, x := range results {
		sb.WriteString(x + "\n")
	}
	if len(results) == 0 {
		sb.WriteString("无\n")
	}
	sb.WriteString("\n[错误]\n")
	for _, x := range errs {
		sb.WriteString(x + "\n")
	}
	if len(errs) == 0 {
		sb.WriteString("无\n")
	}
	sb.WriteString("\n说明：Face Fix 是可选修复。静态检测只能发现可疑绑定，不能证明游戏内一定出现白圈或妆容异常。共享/复杂 CommandList 不会自动改写。\n")
	logPath, logErr := outputLog(root, now, sb.String())
	msg := fmt.Sprintf("疑似风险 %d 条；可修改 INI %d 个；已修改 %d 个；人工检查 %d 处。", len(findings), len(plans), modified, len(manual))
	if preview {
		msg += "\n本次未修改文件。"
	}
	if logErr != nil {
		msg += "\n日志写入失败：" + logErr.Error()
	} else {
		msg += "\n日志：" + filepath.Base(logPath)
	}
	notify("ORFix Repair v3.4 — Face Fix 结果", msg, logErr == nil && len(errs) == 0)
	if runtime.GOOS != "windows" {
		fmt.Println(sb.String())
	}
}
