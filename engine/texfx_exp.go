package engine

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	expTexRun  = regexp.MustCompile(`(?i)^\s*run\s*=\s*CommandList\\TexFx\\Transparency(?:\.[01])?\s*(?:;.*)?$`)
	expTex69   = regexp.MustCompile(`(?i)^\s*ps-t69\s*=\s*(Resource[A-Za-z0-9_.-]+)\s*(?:;.*)?$`)
	expDraw    = regexp.MustCompile(`(?i)^(\s*)drawindexed\s*=\s*([0-9]+)\s*,\s*([0-9]+)\s*,\s*([0-9]+)\s*(?:;.*)?$`)
	expComment = regexp.MustCompile(`(?i)(?:(?:^|[^a-z])(?:mesh|ring|net|veil|lace|tulle|sheer|transparen\w*|stocking\w*|glasses)(?:[^a-z]|$)|网纱|透明|丝袜|蕾丝)`)
)

// Experimental heuristic. Runs ONLY when user explicitly opts into the riskier
// 'experimental-texfx' mode. This is not a reliable replacement for UV analysis;
// the target draw segments are inferred from labels near drawindexed calls.
func transformExperimentalTexFx(path string, content []byte) ([]byte, []string, []string) {
	if isLibraryNamespace(content) || bytes.IndexByte(content, 0) >= 0 {
		return nil, nil, nil
	}
	lines := strings.Split(string(content), "\n")
	ss := parseSections(lines)
	res := resourceFiles(lines, ss)
	edits := map[int]string{}
	var changes, manual []string
	for _, s := range ss {
		n := strings.ToLower(s.name)
		if !strings.HasPrefix(n, "textureoverride") || !strings.Contains(n, "body") {
			continue
		}
		var mask string
		maskline, texline := -1, -1
		type candidate struct {
			line                int
			count, offset, base int
			indent, label       string
		}
		var candidates []candidate
		totalDraws := 0
		var comment string
		var activeOR bool
		for i := s.start + 1; i < s.end; i++ {
			val := normLine(lines[i])
			t := strings.TrimSpace(val)
			if strings.HasPrefix(t, ";") {
				comment = t
				continue
			}
			if m := expTex69.FindStringSubmatch(val); m != nil {
				mask = m[1]
				maskline = i
			}
			if expTexRun.MatchString(val) {
				texline = i
			}
			if extFix.MatchString(val) {
				activeOR = true
			}
			if m := expDraw.FindStringSubmatch(val); m != nil {
				totalDraws++
				count, _ := strconv.Atoi(m[2])
				offset, _ := strconv.Atoi(m[3])
				base, _ := strconv.Atoi(m[4])
				if count > 0 && expComment.MatchString(comment) {
					candidates = append(candidates, candidate{i, count, offset, base, m[1], comment})
				}
			}
			if t != "" && !strings.HasPrefix(t, ";") {
				comment = ""
			}
		}
		if texline < 0 || maskline < 0 || mask == "" || !fileForRef(path, mask, res) || totalDraws < 3 || len(candidates) == 0 || len(candidates) >= totalDraws {
			if texline >= 0 && totalDraws >= 3 {
				manual = append(manual, fmt.Sprintf("%s: 检测到分段透明渲染，但没有足够可靠的注释定位，未改", s.name))
			}
			continue
		}
		// Earlier ORFix/NNFix is preserved. This transformation only changes a
		// single TexFx pass into component draws for named semi-transparent meshes.
		_ = activeOR
		old := normLine(lines[maskline])
		indent := old[:len(old)-len(strings.TrimLeft(old, " \t"))]
		edits[maskline] = indent + "ps-t69 = null"
		edits[texline] = indent + "; ORFix Repair experimental: global transparency replaced by named component passes"
		for _, c := range candidates {
			block := []string{
				c.indent + "ps-t69 = " + mask,
				fmt.Sprintf("%s$\\TexFx\\_1 = %d", c.indent, c.count),
				fmt.Sprintf("%s$\\TexFx\\_2 = %d", c.indent, c.offset),
				fmt.Sprintf("%s$\\TexFx\\_3 = %d", c.indent, c.base),
				c.indent + `run = CommandList\TexFx\Component`,
				c.indent + "ps-t69 = null",
			}
			edits[c.line] = strings.Join(block, "\n")
		}
		changes = append(changes, fmt.Sprintf("%s：[实验 TexFx] 原遮罩 %s，按备注匹配 %d/%d 条 drawindexed（无需改 DDS；需游戏验证）", s.name, mask, len(candidates), totalDraws))
	}
	if len(changes) == 0 {
		return nil, nil, manual
	}
	end := ""
	if bytes.Contains(content, []byte("\r\n")) {
		end = "\r"
	}
	out := make([]string, 0, len(lines)+len(changes)*5)
	for i, line := range lines {
		if v, ok := edits[i]; ok {
			if end != "" {
				v = strings.ReplaceAll(v, "\n", "\r\n") + end
			}
			out = append(out, v)
		} else {
			out = append(out, line)
		}
	}
	return []byte(strings.Join(out, "\n")), changes, manual
}

var _ = filepath.Separator
