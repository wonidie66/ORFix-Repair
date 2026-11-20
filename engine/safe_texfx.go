package engine

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

var safeTexRun = regexp.MustCompile(`(?i)^\s*run\s*=\s*CommandList\\TexFx\\Transparency(?:\.[01])?\s*(?:;.*)?$`)
var safeMask = regexp.MustCompile(`(?i)^\s*ps-t69\s*=\s*(Resource[A-Za-z0-9_.-]+)\s*(?:;.*)?$`)

// Add only the harmless NNFix call to a verified, unbranched, standard
// two-texture + one ps-t69 transparency pass. Never change the mask, shader
// version, draw calls, or TexFx texture data.
func transformSafeTexFx(path string, content []byte) ([]byte, []string, []string) {
	if bytes.IndexByte(content, 0) >= 0 || isLibraryNamespace(content) {
		return nil, nil, nil
	}
	lines := strings.Split(string(content), "\n")
	ss := parseSections(lines)
	refs := resourceFiles(lines, ss)
	after := map[int][]string{}
	var changes, manual []string
	for _, s := range ss {
		if !validSectionName(s.name) || !isCharacterOutfitCandidate(s.name, ss) || !hasDrawBinding(lines, s) {
			continue
		}
		if containsConditional(lines, s) {
			continue
		}
		var b []binding
		maskRef := ""
		maskLine := -1
		texRunCount := 0
		otherRun := false
		strangeSlots := false
		already := false
		for i := s.start + 1; i < s.end; i++ {
			line := normLine(lines[i])
			t := strings.TrimSpace(line)
			if t == "" || strings.HasPrefix(t, ";") {
				continue
			}
			if extFix.MatchString(line) {
				already = true
				continue
			}
			if m := safeMask.FindStringSubmatch(line); m != nil {
				maskRef = m[1]
				maskLine = i
				continue
			}
			if m := extResource.FindStringSubmatch(line); m != nil {
				switch m[1] {
				case "0", "1", "2":
					slot := int(m[1][0] - '0')
					b = append(b, binding{line: i, slot: slot, ref: m[2]})
				default:
					strangeSlots = true
				}
				continue
			}
			if strings.HasPrefix(strings.ToLower(t), "ps-t") {
				strangeSlots = true
			}
			if safeTexRun.MatchString(line) {
				texRunCount++
				continue
			}
			if extAnyRun.MatchString(line) {
				otherRun = true
			}
		}
		if texRunCount == 0 || already {
			continue
		}
		if otherRun || strangeSlots || texRunCount != 1 || len(b) != 2 || maskLine < 0 || !fileForRef(path, maskRef, refs) {
			manual = append(manual, s.name+"：[TexFx] 透明配置不满足严格自动修复条件，保留原样")
			continue
		}
		kind, reason := classifyBindings(path, b, refs)
		if kind != "NNFix" {
			manual = append(manual, s.name+"：[TexFx] 贴图布局不明确（"+reason+"），保留原样")
			continue
		}
		anchor := maskLine
		if b[1].line > anchor {
			anchor = b[1].line
		}
		after[anchor] = append(after[anchor], nnfixRun)
		changes = append(changes, fmt.Sprintf("%s：[NNFix+TexFx] 已验证双贴图和透明遮罩，在透明渲染前添加 NNFix（保留 ps-t69 及 Transparency 原版本）", s.name))
	}
	if len(after) == 0 {
		return nil, nil, manual
	}
	suffix := ""
	if bytes.Contains(content, []byte("\r\n")) {
		suffix = "\r"
	}
	output := make([]string, 0, len(lines)+len(changes))
	for i, line := range lines {
		output = append(output, line)
		for _, added := range after[i] {
			output = append(output, added+suffix)
		}
	}
	return []byte(strings.Join(output, "\n")), changes, manual
}
