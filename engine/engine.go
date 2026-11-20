package engine

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const maxINIBytes = 8 * 1024 * 1024
const orfixRun = `run = CommandList\global\ORFix\ORFix`
const nnfixRun = `run = CommandList\global\ORFix\NNFix`

var (
	reHeader       = regexp.MustCompile(`^\s*\[([^\]\r\n]+)\]\s*(?:;.*)?$`)
	reNameToken    = regexp.MustCompile(`[A-Z]?[a-z]+|[A-Z]+|[0-9]+`)
	reTex          = regexp.MustCompile(`(?i)^([ \t]*)(;[ \t]*)?ps-t([0-9]+)[ \t]*=[ \t]*(Resource[A-Za-z0-9_.-]+)[ \t]*(?:;.*)?$`)
	reFilename     = regexp.MustCompile(`(?i)^\s*filename\s*=\s*(\S(?:.*\S)?)\s*$`)
	reMatchFirst   = regexp.MustCompile(`(?i)^\s*match_first_index\s*=`)
	reIB           = regexp.MustCompile(`(?i)^\s*ib\s*=`)
	reRunOR        = regexp.MustCompile(`(?i)^\s*run\s*=\s*commandlist\\global\\orfix\\(orfix|nnfix)\s*(?:;.*)?$`)
	reConditional  = regexp.MustCompile(`(?i)^\s*(if\s|else\b|elif\s|endif\b)`)
	reGlobalNS     = regexp.MustCompile(`(?im)^\s*namespace\s*=\s*global(?:\\|\s*$)`)
	reAnyNamespace = regexp.MustCompile(`(?im)^\s*namespace\s*=`)
	reTexFxRun     = regexp.MustCompile(`(?im)^\s*run\s*=\s*CommandList\\TexFx\\`)
	reAnyRun       = regexp.MustCompile(`(?im)^\s*run\s*=`)
)

type section struct {
	name       string
	start, end int
}
type texEntry struct {
	line      int
	ref       string
	commented bool
}
type update struct {
	line         int
	replacement  string
	insertBefore []string
	insertAfter  []string
}
type plan struct {
	path       string
	original   []byte
	digest     [32]byte
	patched    []byte
	changes    []string
	manual     []string
	companions []plan
}
type scanReport struct {
	root         string
	scanned      int
	skipped      int
	repairs      []plan
	manual       []string
	errors       []string
	globals      []string
	findings     []string
	warnings     []string
	orfixFound   bool
	texfxFound   bool
	usesORFix    bool
	usesTexFx    bool
	faceFindings []string
}

func normLine(s string) string { return strings.TrimSuffix(s, "\r") }
func parseSections(lines []string) []section {
	var ss []section
	for i, raw := range lines {
		v := normLine(raw)
		if m := reHeader.FindStringSubmatch(v); m != nil {
			if len(ss) > 0 {
				ss[len(ss)-1].end = i
			}
			ss = append(ss, section{name: strings.TrimSpace(m[1]), start: i, end: len(lines)})
		}
	}
	return ss
}
func resourceFiles(lines []string, sections []section) map[string]string {
	refs := make(map[string]string)
	for _, sec := range sections {
		if !strings.HasPrefix(strings.ToLower(sec.name), "resource") {
			continue
		}
		for i := sec.start + 1; i < sec.end; i++ {
			if m := reFilename.FindStringSubmatch(normLine(lines[i])); m != nil {
				fname := strings.TrimSpace(strings.SplitN(m[1], ";", 2)[0])
				fname = strings.Trim(fname, `"`)
				if fname != "" {
					refs[strings.ToLower(sec.name)] = fname
				}
				break
			}
		}
	}
	return refs
}
func fileForRef(iniPath, ref string, resources map[string]string) bool {
	name, ok := resources[strings.ToLower(ref)]
	if !ok || name == "" {
		return false
	}
	name = strings.ReplaceAll(name, `\`, string(os.PathSeparator))
	if filepath.IsAbs(name) {
		return false
	}
	full := filepath.Join(filepath.Dir(iniPath), name)
	st, err := os.Stat(full)
	return err == nil && st.Mode().IsRegular() && st.Size() > 0
}
func texBase(ref, kind string) (string, bool) {
	s := strings.ToLower(ref)
	if !strings.HasSuffix(s, kind) {
		return "", false
	}
	return ref[:len(ref)-len(kind)], true
}
func validSectionName(n string) bool {
	if !strings.HasPrefix(strings.ToLower(n), "textureoverride") {
		return false
	}
	// Match complete CamelCase tokens instead of substrings. The old
	// strings.Contains("gun") rule incorrectly rejected RaidenShogunHead
	// because "Shogun" ends with the letters "gun".
	tail := n[len("TextureOverride"):]
	tokens := reNameToken.FindAllString(tail, -1)
	blocked := map[string]bool{
		"face": true, "gun": true, "weapon": true, "icon": true,
		"shadow": true, "outline": true, "effect": true,
		"eyebrow": true, "pupil": true, "glow": true, "fix": true,
	}
	for _, token := range tokens {
		if blocked[strings.ToLower(token)] {
			return false
		}
	}
	// Conservative fallback for all-lowercase legacy names, where CamelCase
	// boundaries are unavailable. Do not use the ambiguous substring "gun".
	if len(tokens) <= 1 {
		lower := strings.ToLower(tail)
		for _, bad := range []string{"face", "weapon", "icon", "shadow", "outline", "effect", "eyebrow", "pupil", "glow", "fix"} {
			if strings.Contains(lower, bad) {
				return false
			}
		}
	}
	return true
}

// Head-only exports can also be weapons (e.g. SacrificialFragmentsHead).
// For such ambiguous INIs, only auto-process Head if an outfit counterpart
// with the same stem (Body, Dress or Extra) is present.
func isCharacterOutfitCandidate(name string, all []section) bool {
	n := strings.ToLower(name)
	prefix := "textureoverride"
	if !strings.HasPrefix(n, prefix) {
		return false
	}
	base := strings.TrimPrefix(n, prefix)
	// Only common character mesh sections qualify; arbitrary object names are unsafe.
	kinds := []string{"head", "body", "dress", "extra"}
	kind := ""
	for _, x := range kinds {
		if strings.HasSuffix(base, x) {
			kind = x
			break
		}
	}
	if kind == "" {
		return false
	}
	stem := strings.TrimSuffix(base, kind)
	// Main character meshes: Body/Dress/Extra are accepted directly; Head must
	// have a sibling body/dress/extra so an isolated weapon Head is not mistaken
	// for a character mesh.
	if kind != "head" {
		return true
	}
	if hasMainOutfitStem(stem, all) {
		return true
	}
	// Independent character submeshes such as EyeHead and HairHead are valid
	// when their parent character has a Head + Body/Dress/Extra pair. Their
	// IB/DDS files are verified separately before any patch is applied.
	for _, token := range []string{"eyes", "eye", "hair"} {
		if strings.HasSuffix(stem, token) {
			parent := strings.TrimSuffix(stem, token)
			if parent != "" && hasMainOutfitStem(parent, all) {
				return true
			}
		}
	}
	return false
}

func hasMainOutfitStem(stem string, all []section) bool {
	prefix := "textureoverride" + strings.ToLower(stem)
	parts := map[string]bool{}
	for _, sec := range all {
		other := strings.ToLower(sec.name)
		if strings.HasPrefix(other, prefix) {
			suffix := strings.TrimPrefix(other, prefix)
			if suffix == "head" || suffix == "body" || suffix == "dress" || suffix == "extra" {
				parts[suffix] = true
			}
		}
	}
	return parts["head"] && (parts["body"] || parts["dress"] || parts["extra"])
}

func isCharacterSubcomponentHead(name string, all []section) bool {
	n := strings.ToLower(name)
	if !strings.HasPrefix(n, "textureoverride") || !strings.HasSuffix(n, "head") {
		return false
	}
	stem := strings.TrimSuffix(strings.TrimPrefix(n, "textureoverride"), "head")
	for _, token := range []string{"eyes", "eye", "hair"} {
		if strings.HasSuffix(stem, token) {
			parent := strings.TrimSuffix(stem, token)
			return parent != "" && hasMainOutfitStem(parent, all)
		}
	}
	return false
}

func sectionIBRef(lines []string, sec section) string {
	for i := sec.start + 1; i < sec.end; i++ {
		if m := regexp.MustCompile(`(?i)^\s*ib\s*=\s*(Resource[A-Za-z0-9_.-]+)`).FindStringSubmatch(normLine(lines[i])); m != nil {
			return m[1]
		}
	}
	return ""
}

func verifiedSubcomponentIB(path string, lines []string, sec section, resources map[string]string) bool {
	ref := sectionIBRef(lines, sec)
	return ref != "" && fileForRef(path, ref, resources)
}

func hasLegacyReflection(data []byte) bool {
	t := strings.ToLower(string(data))
	return strings.Contains(t, "[shaderregexcharreflection]") || strings.Contains(t, "[commandlistreflectiontexture]")
}
func containsConditional(lines []string, sec section) bool {
	for i := sec.start + 1; i < sec.end; i++ {
		if reConditional.MatchString(normLine(lines[i])) {
			return true
		}
	}
	return false
}
func hasDrawBinding(lines []string, sec section) bool {
	for i := sec.start + 1; i < sec.end; i++ {
		x := normLine(lines[i])
		if reMatchFirst.MatchString(x) || reIB.MatchString(x) {
			return true
		}
	}
	return false
}

// simpleTransparencyShader confirms a legacy fixed-factor transparent draw
// whose stock index draw can safely be preceded by the NNFix texture-slot fix.
// Do NOT apply this to arbitrary CustomShaders, TexFx, complex commands, or
// conditional component lists.
func simpleTransparencyShader(lines []string, sections []section, mesh section) bool {
	runCount := 0
	for i := mesh.start + 1; i < mesh.end; i++ {
		v := strings.ToLower(strings.TrimSpace(normLine(lines[i])))
		if !strings.HasPrefix(v, "run") {
			continue
		}
		if !strings.HasPrefix(v, "run ") && !strings.HasPrefix(v, "run=") && !strings.HasPrefix(v, "run\t") {
			continue
		}
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return false
		}
		cmd := strings.TrimSpace(strings.SplitN(parts[1], ";", 2)[0])
		if cmd != "customshadertransparency" {
			return false
		}
		runCount++
	}
	if runCount != 1 {
		return false
	}

	for _, shader := range sections {
		if !strings.EqualFold(shader.name, "CustomShaderTransparency") {
			continue
		}
		var blend, handling, drawing bool
		for i := shader.start + 1; i < shader.end; i++ {
			v := strings.ToLower(strings.TrimSpace(normLine(lines[i])))
			if v == "" || strings.HasPrefix(v, ";") {
				continue
			}
			switch {
			case v == "blend = add blend_factor inv_blend_factor", v == "blend=add blend_factor inv_blend_factor":
				blend = true
			case v == "handling = skip", v == "handling=skip":
				handling = true
			case v == "drawindexed = auto", v == "drawindexed=auto":
				drawing = true
			case strings.HasPrefix(v, "blend_factor["):
				// Preserve each original parameter. They are not modified.
			default:
				// Unrecognized shader setup could reroute draw calls or slots.
				return false
			}
		}
		return blend && handling && drawing
	}
	return false
}

// transformINI patches only predictable three-texture character mesh overrides.
// It never changes the original text's encoding, line endings, or unrelated sections.
func transformINI(path string, content []byte) ([]byte, []string, []string) {
	// Exact-match, game-verified recipe is checked before generic heuristics.
	if patched, changes, ok := knownRecipe(path, content); ok {
		return patched, changes, nil
	}
	if isKnownPatchedCitlali(path, content) {
		return nil, nil, nil
	}
	if bytes.IndexByte(content, 0) >= 0 {
		return nil, nil, []string{"跳过：INI 含二进制 NUL 数据"}
	}
	if reAnyNamespace.Match(content) || strings.EqualFold(filepath.Base(path), "ORFix.ini") {
		return nil, nil, nil // namespaced libraries or complex namespaced overrides are not modified
	}
	if hasLegacyReflection(content) {
		return nil, nil, []string{"含旧式 Reflection ShaderRegex/CommandList；为防冲突，未自动改写"}
	}
	lines := strings.Split(string(content), "\n")
	sections := parseSections(lines)
	resources := resourceFiles(lines, sections)
	// A patch to a given input line may have inserts before or after it.
	replacements := make(map[int]string)
	before := make(map[int][]string)
	after := make(map[int][]string)
	var changes, manual []string
	newline := "\n"
	if bytes.Contains(content, []byte("\r\n")) {
		newline = "\r\n"
	}
	_ = newline // Existing line chunks retain their endings; insertions are normalized below.

	for _, sec := range sections {
		if !validSectionName(sec.name) {
			continue
		}
		if !hasDrawBinding(lines, sec) {
			continue
		}
		if !isCharacterOutfitCandidate(sec.name, sections) {
			manual = append(manual, sec.name+"：孤立 Head/非常规名称，可能是武器或道具，安全起见跳过")
			continue
		}
		if isCharacterSubcomponentHead(sec.name, sections) && !verifiedSubcomponentIB(path, lines, sec, resources) {
			manual = append(manual, sec.name+"：识别为独立眼睛/头发子网格，但 IB 资源文件无法核实，跳过自动修改")
			continue
		}
		if containsConditional(lines, sec) {
			if reTexFxRun.MatchString(strings.Join(lines[sec.start+1:sec.end], "\n")) {
				manual = append(manual, sec.name+"：[TexFx] 存在透明调用及复杂网格/条件绘制，须手动检查（未知透明材质不自动修改）")
			} else {
				manual = append(manual, sec.name+"：含 if/else 等条件分支，暂不自动修复")
			}
			continue
		}
		entries := map[int][]texEntry{}
		var existingOR bool
		var existingNN bool
		var otherSlots bool
		for i := sec.start + 1; i < sec.end; i++ {
			v := normLine(lines[i])
			if r := reRunOR.FindStringSubmatch(v); r != nil {
				if strings.EqualFold(r[1], "orfix") {
					existingOR = true
				} else {
					existingNN = true
				}
			}
			if m := reTex.FindStringSubmatch(v); m != nil {
				slot := -1
				switch m[3] {
				case "0":
					slot = 0
				case "1":
					slot = 1
				case "2":
					slot = 2
				default:
					otherSlots = true
				}
				if slot >= 0 {
					entries[slot] = append(entries[slot], texEntry{line: i, ref: m[4], commented: m[2] != ""})
				}
			} else if strings.HasPrefix(strings.ToLower(strings.TrimSpace(v)), "ps-t") {
				// Unrecognized texture format is not safe to rewrite.
				otherSlots = true
			}
		}
		if otherSlots || existingNN {
			if otherSlots && reTexFxRun.MatchString(strings.Join(lines[sec.start+1:sec.end], "\n")) {
				manual = append(manual, sec.name+"：[TexFx] 额外槽位和透明命令需按网格检查，未自动修改")
			} else if !existingNN {
				manual = append(manual, sec.name+"：额外/复杂贴图槽位，未自动修改")
			}
			continue
		}
		if len(entries[2]) != 1 || entries[2][0].commented {
			// Two-texture auto-fix: standard unambiguous Diffuse + LightMap,
			// precisely one live resource in each slot, no extra bindings,
			// no branching/legacy commands, and actual DDS files on disk.
			if len(entries[2]) == 0 && len(entries[0]) == 1 && len(entries[1]) == 1 &&
				!entries[0][0].commented && !entries[1][0].commented && !existingOR && !existingNN {
				diff := entries[0][0].ref
				light := entries[1][0].ref
				stem, ok := texBase(diff, "diffuse")
				transparentDraw := simpleTransparencyShader(lines, sections, sec)
				noOtherRun := !reAnyRun.MatchString(strings.Join(lines[sec.start+1:sec.end], "\n"))
				if ok && strings.EqualFold(stem+"LightMap", light) &&
					fileForRef(path, diff, resources) && fileForRef(path, light, resources) &&
					(noOtherRun || transparentDraw) {
					// If the mesh has an IB, verify it is an existing file as well.
					// Some old exports have match_first_index alone; that remains valid.
					insAt := entries[1][0].line
					after[insAt] = append(after[insAt], nnfixRun)
					if transparentDraw {
						changes = append(changes, sec.name+"：[NNFix] 已识别固定混合半透明 CustomShaderTransparency，在其执行前插入 NNFix 并保留透明绘制")
					} else {
						changes = append(changes, sec.name+"：[NNFix] 双贴图部件已接入 NNFix")
					}
				} else {
					manual = append(manual, sec.name+"：双贴图存在不确定性（资源缺失、其他 CommandList 或命名不对应），未自动修改")
				}
			} else if len(entries[2]) == 0 && (len(entries[0]) > 0 || len(entries[1]) > 0) {
				manual = append(manual, sec.name+"：双贴图不完整或已有修复，未自动修改")
			}
			continue
		}
		light := entries[2][0]
		base, isLM := texBase(light.ref, "lightmap")
		if !isLM {
			continue
		}
		// For strict resource matching, require a *common stem* for all three maps.
		normalRef, diffuseRef := base+"NormalMap", base+"Diffuse"
		if !fileForRef(path, light.ref, resources) || !fileForRef(path, normalRef, resources) || !fileForRef(path, diffuseRef, resources) {
			manual = append(manual, sec.name+"：找不到三张贴图资源定义或实际文件，未自动修改")
			continue
		}
		expected := []string{normalRef, diffuseRef}
		inserts := []string{}
		restores := []int{}
		safe := true
		for slot := 0; slot <= 1; slot++ {
			these := entries[slot]
			// No more than one assignment per slot unless the repeated one is a matching comment.
			active := []texEntry{}
			commented := []texEntry{}
			for _, e := range these {
				if e.commented {
					commented = append(commented, e)
				} else {
					active = append(active, e)
				}
			}
			if len(active) > 1 || len(commented) > 1 || (len(active) > 0 && len(commented) > 0 && !strings.EqualFold(active[0].ref, commented[0].ref)) {
				safe = false
				break
			}
			if len(active) > 0 {
				if !strings.EqualFold(active[0].ref, expected[slot]) {
					safe = false
					break
				}
			} else if len(commented) > 0 {
				if !strings.EqualFold(commented[0].ref, expected[slot]) {
					safe = false
					break
				}
				restores = append(restores, commented[0].line)
			} else {
				inserts = append(inserts, fmt.Sprintf("ps-t%d = %s", slot, expected[slot]))
			}
		}
		if !safe {
			manual = append(manual, sec.name+"：贴图名称或槽位冲突，未自动修改")
			continue
		}
		// Existing normal/diffuse maps and no ORFix -> add ORFix only.
		if len(inserts) == 0 && len(restores) == 0 && existingOR {
			continue
		}
		if len(inserts) > 0 {
			// Missing one or both slots is considerably more ambiguous than a commented line.
			// Automatic restoration is allowed only when at least one of t0/t1 is explicit,
			// and both resources exist on disk.
			if len(entries[0])+len(entries[1]) == 0 {
				manual = append(manual, sec.name+"：ps-t0/ps-t1 均缺失，不能排除作者有意留空")
				continue
			}
		}
		for _, ln := range restores {
			raw := lines[ln]
			lf := strings.HasSuffix(raw, "\r")
			if lf {
				raw = strings.TrimSuffix(raw, "\r")
			}
			prefixLen := len(raw) - len(strings.TrimLeft(raw, " \t"))
			p := raw[:prefixLen]
			remain := strings.TrimLeft(raw[prefixLen:], "; \t")
			if lf {
				remain += "\r"
			}
			replacements[ln] = p + remain
		}
		if len(inserts) > 0 {
			t2line := lines[light.line]
			ws := t2line[:len(t2line)-len(strings.TrimLeft(t2line, " \t"))]
			for _, ins := range inserts {
				before[light.line] = append(before[light.line], ws+ins)
			}
		}
		if !existingOR {
			after[light.line] = append(after[light.line], orfixRun)
		}
		what := []string{}
		if len(restores) > 0 {
			what = append(what, fmt.Sprintf("恢复 %d 个被注释的贴图槽位", len(restores)))
		}
		if len(inserts) > 0 {
			what = append(what, fmt.Sprintf("补充 %d 个缺失的贴图槽位", len(inserts)))
		}
		if !existingOR {
			what = append(what, "接入 ORFix")
		}
		changes = append(changes, sec.name+"：[ORFix] "+strings.Join(what, "；"))
	}
	if len(changes) == 0 {
		return nil, nil, manual
	}
	// Preserve CRLF by adding the original line-ending style to added lines.
	suffix := ""
	if bytes.Contains(content, []byte("\r\n")) {
		suffix = "\r"
	}
	out := make([]string, 0, len(lines)+len(changes)*3)
	for i, line := range lines {
		for _, v := range before[i] {
			out = append(out, v+suffix)
		}
		if v, ok := replacements[i]; ok {
			line = v
		}
		out = append(out, line)
		for _, v := range after[i] {
			out = append(out, v+suffix)
		}
	}
	return []byte(strings.Join(out, "\n")), changes, manual
}

func scan(root string, modes ...string) scanReport {
	mode := "safe"
	if len(modes) > 0 {
		mode = modes[0]
	}
	report := scanReport{root: root}
	var ningguangINIs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			report.errors = append(report.errors, path+"："+walkErr.Error())
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) != ".ini" {
			return nil
		}
		report.scanned++
		if strings.EqualFold(d.Name(), "Ningguang.ini") {
			ningguangINIs = append(ningguangINIs, path)
		}
		if strings.HasPrefix(strings.ToLower(d.Name()), "disabled") {
			report.skipped++
			return nil
		}
		st, err := d.Info()
		if err != nil {
			report.errors = append(report.errors, path+"："+err.Error())
			return nil
		}
		if st.Size() > maxINIBytes {
			report.skipped++
			report.manual = append(report.manual, path+"：文件过大，跳过")
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			report.errors = append(report.errors, path+"："+err.Error())
			return nil
		}
		if isORFixLibrary(path, content) {
			report.orfixFound = true
		}
		if isTexFxLibrary(path, content) {
			report.texfxFound = true
		}
		lowerContent := strings.ToLower(string(content))
		if strings.Contains(lowerContent, `commandlist\global\orfix\`) {
			report.usesORFix = true
		}
		if strings.Contains(lowerContent, `commandlist\texfx\`) || strings.Contains(lowerContent, "ps-t69") {
			report.usesTexFx = true
		}
		if !strings.EqualFold(d.Name(), "ORFix.ini") && !isLibraryNamespace(content) {
			for _, finding := range diagnoseINI(path, content) {
				report.findings = append(report.findings, rel(root, path)+" | "+finding)
			}
			faceCheck := analyzeFaceINI(path, content)
			for _, finding := range faceCheck.findings {
				report.faceFindings = append(report.faceFindings, rel(root, path)+" | "+finding)
			}
			for _, manual := range faceCheck.manual {
				report.manual = append(report.manual, rel(root, path)+" | [Face Fix 人工检查] "+manual)
			}
		}
		if strings.EqualFold(d.Name(), "ORFix.ini") || (mode == "safe" && reAnyNamespace.Match(content)) || (mode != "safe" && isLibraryNamespace(content)) {
			report.globals = append(report.globals, path)
			report.skipped++
			return nil
		}
		var patched []byte
		var changes, manual []string
		var companions []plan
		if mode == "safe" {
			patched, changes, manual = transformINI(path, content)
		} else {
			working := content
			migrated, companion, legacyChanges, legacyManual := migrateNingguangLegacy(path, content)
			manual = append(manual, legacyManual...)
			if len(migrated) > 0 && companion != nil {
				working = migrated
				companions = append(companions, *companion)
				changes = append(changes, legacyChanges...)
			}
			// Strict, verified TexFx+NNFix handling is part of normal enhanced mode.
			texfxSafe, texChanges, texManual := transformSafeTexFx(path, working)
			if len(texfxSafe) > 0 {
				working = texfxSafe
				changes = append(changes, texChanges...)
			}
			manual = append(manual, texManual...)
			enhanced, standardChanges, standardManual := transformEnhanced(path, working, "enhanced")
			if len(enhanced) > 0 {
				working = enhanced
				changes = append(changes, standardChanges...)
			}
			manual = append(manual, standardManual...)
			if mode == "experimental" {
				texfx, xchanges, xmanual := transformExperimentalTexFx(path, working)
				if len(texfx) > 0 {
					working = texfx
					changes = append(changes, xchanges...)
				}
				manual = append(manual, xmanual...)
			}
			if !bytes.Equal(working, content) {
				patched = working
			}
		}
		if len(changes) > 0 && bytes.Contains(patched, []byte(`CommandList\global\ORFix\`)) {
			report.usesORFix = true
		}
		for _, m := range manual {
			// v3.1 could report "TexFx skipped" even after an earlier standard
			// NNFix+TexFx pass already fixed that exact section. Preserve all
			// other warnings, but do not contradict an applied safe fix.
			resolved := false
			for _, c := range changes {
				if strings.Contains(c, "[NNFix+TexFx]") && strings.HasPrefix(m, strings.SplitN(c, "：", 2)[0]+"：") && strings.Contains(m, "[TexFx]") {
					resolved = true
					break
				}
			}
			if !resolved {
				report.manual = append(report.manual, path+" | "+m)
			}
		}
		if len(changes) > 0 && !bytes.Equal(patched, content) {
			report.repairs = append(report.repairs, plan{path: path, original: content, digest: sha256.Sum256(content), patched: patched, changes: changes, manual: manual, companions: companions})
		}
		return nil
	})
	if err != nil {
		report.errors = append(report.errors, err.Error())
	}
	if len(ningguangINIs) > 1 {
		report.warnings = append(report.warnings, fmt.Sprintf("发现 %d 个凝光版本配置；同时启用多个替换同一角色的 Mod 可能冲突，建议游戏测试时只启用一个。", len(ningguangINIs)))
	}
	if !report.orfixFound {
		report.orfixFound = findLibraryInAncestors(root, true)
	}
	if !report.texfxFound {
		report.texfxFound = findLibraryInAncestors(root, false)
	}
	if report.usesORFix && !report.orfixFound {
		report.warnings = append(report.warnings, "扫描范围内未发现 ORFix 全局库（namespace = global\\ORFix 或 ORFix.ini）。若全局库安装在扫描目录之外，请忽略；否则 NNFix/ORFix 调用可能无法生效。")
	}
	if report.usesTexFx && !report.texfxFound {
		report.warnings = append(report.warnings, "扫描范围内未发现 TexFx 全局库。若 Mod 使用 TexFx，请确认已安装对应版本；若库位于扫描目录之外，请忽略。")
	}
	sort.Slice(report.repairs, func(i, j int) bool { return report.repairs[i].path < report.repairs[j].path })
	return report
}

// backupAndReplace makes a timestamped, exclusive-copy backup before atomic replacement.
// It checks a SHA-256 digest again to avoid overwriting files modified after the scan.
func backupAndReplace(p plan, now time.Time) (string, error) {
	current, err := os.ReadFile(p.path)
	if err != nil {
		return "", err
	}
	if sha256.Sum256(current) != p.digest {
		return "", errors.New("扫描后文件发生变化，已跳过")
	}
	info, err := os.Stat(p.path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("不是普通文件")
	}
	if info.Mode().Perm()&0200 == 0 {
		return "", errors.New("文件为只读，已跳过")
	}
	backupStem := p.path + "_" + now.Format("2006-01-02_15-04-05")
	backupPath := ""
	for i := 0; i < 1000; i++ {
		candidate := backupStem + ".bak"
		if i > 0 {
			candidate = fmt.Sprintf("%s_%03d.bak", backupStem, i+1)
		}
		f, err := os.OpenFile(candidate, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("创建备份失败：%w", err)
		}
		written, werr := f.Write(current)
		if werr == nil && written != len(current) {
			werr = io.ErrShortWrite
		}
		if werr == nil {
			werr = f.Sync()
		}
		closeErr := f.Close()
		if werr == nil {
			werr = closeErr
		}
		if werr != nil {
			os.Remove(candidate)
			return "", fmt.Errorf("写入备份失败：%w", werr)
		}
		backupPath = candidate
		break
	}
	if backupPath == "" {
		return "", errors.New("备份文件名冲突过多")
	}
	tmp, err := os.CreateTemp(filepath.Dir(p.path), ".orfix-repair-*.tmp")
	if err != nil {
		return backupPath, fmt.Errorf("无法创建临时文件：%w", err)
	}
	tempName := tmp.Name()
	defer os.Remove(tempName)
	n, werr := tmp.Write(p.patched)
	if werr == nil && n != len(p.patched) {
		werr = io.ErrShortWrite
	}
	if werr == nil {
		werr = tmp.Sync()
	}
	if werr == nil {
		werr = tmp.Chmod(info.Mode().Perm())
	}
	closeErr := tmp.Close()
	if werr == nil {
		werr = closeErr
	}
	if werr != nil {
		return backupPath, fmt.Errorf("写入临时文件失败：%w", werr)
	}
	if err := os.Rename(tempName, p.path); err != nil {
		return backupPath, fmt.Errorf("替换文件失败（原文件未被删除，备份保留）：%w", err)
	}
	return backupPath, nil
}
