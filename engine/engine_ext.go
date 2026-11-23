package engine

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Enhanced repair is DATAFLOW oriented: follows TextureOverride -> CommandList
// and processes each independent texture binding sequence, including branches
// and hand-written drawindexed blocks. The three core texture streams remain
// unchanged; only the ORFix/NNFix call and optional TexFx t69 reset are added.
// Verified material-layout classifiers run within this generic pass.
var (
	extNamespace = regexp.MustCompile(`(?im)^\s*namespace\s*=\s*([^;\r\n]+)`)
	extResource  = regexp.MustCompile(`(?i)^\s*ps-t([0-9]+)\s*=\s*(Resource[A-Za-z0-9_.-]+)\s*(?:;.*)?$`)
	extRun       = regexp.MustCompile(`(?i)^\s*run\s*=\s*(CommandList[A-Za-z0-9_.-]+)\s*(?:;.*)?$`)
	extAnyRun    = regexp.MustCompile(`(?i)^\s*run\s*=`)
	extDraw      = regexp.MustCompile(`(?i)^\s*(drawindexed|drawindexedinstanced|draw)\s*=`)
	extBranch    = regexp.MustCompile(`(?i)^\s*(?:if\b|else\b|elif\b|endif\b)`)
	extFix       = regexp.MustCompile(`(?i)^\s*run\s*=\s*CommandList\\global\\ORFix\\(?:NNFix|ORFix)\b`)
	extTex69     = regexp.MustCompile(`(?i)^\s*ps-t69\s*=`)
	extTarget    = regexp.MustCompile(`(?i)(head|body|dress|extra|clothes|skirt|chest|torso|leg|hair|cape|outfit)(?:\.[0-9]+)?$`)
	extCategory  = regexp.MustCompile(`(?i)(head|body|dress|extra|clothes|skirt|chest|torso|leg|hair|cape|outfit)`)
)

type binding struct {
	line, slot int
	ref        string
}
type stagedChange struct {
	pos    int
	lines  []string
	reason string
}

func isLibraryNamespace(content []byte) bool {
	m := extNamespace.FindSubmatch(content)
	if m == nil {
		return false
	}
	ns := strings.ToLower(strings.TrimSpace(string(m[1])))
	return ns == "texfx" || ns == "gimi" || ns == "orfix" || strings.HasPrefix(ns, `global\`) || strings.Contains(ns, "shaderfix")
}

func enhancedCandidate(name string, reachable map[string]bool) bool {
	n := strings.ToLower(name)
	if !reachable[n] {
		return false
	}
	if strings.HasPrefix(n, "commandlist") || strings.HasPrefix(n, "textureoverride") {
		// Check full CamelCase tokens rather than raw substrings; the word
		// "Shogun" contains "gun" but is not a weapon.
		tail := name
		if strings.HasPrefix(n, "textureoverride") {
			tail = name[len("TextureOverride"):]
		}
		if strings.HasPrefix(n, "commandlist") {
			tail = name[len("CommandList"):]
		}
		for _, tok := range reNameToken.FindAllString(tail, -1) {
			switch strings.ToLower(tok) {
			case "face", "weapon", "gun", "sword", "icon", "outline", "shadow", "particle", "weaponfx":
				return false
			}
		}
		return true
	}
	return false
}

func linkedSections(path string, lines []string, ss []section, resources map[string]string) map[string]bool {
	result := map[string]bool{}
	graph := map[string][]string{}
	outfitCandidates := map[string]bool{}
	for _, s := range ss {
		n := strings.ToLower(s.name)
		if !strings.HasPrefix(n, "textureoverride") || !extCategory.MatchString(n) || !validSectionName(s.name) {
			continue
		}
		if !isCharacterOutfitCandidate(s.name, ss) {
			continue
		}
		if isCharacterSubcomponentHead(s.name, ss) && !verifiedSubcomponentIB(path, lines, s, resources) {
			continue
		}
		outfitCandidates[n] = true
	}
	for _, s := range ss {
		n := strings.ToLower(s.name)
		for i := s.start + 1; i < s.end; i++ {
			if m := extRun.FindStringSubmatch(normLine(lines[i])); m != nil {
				graph[n] = append(graph[n], strings.ToLower(m[1]))
			}
		}
	}
	pending := []string{}
	for n := range outfitCandidates {
		result[n] = true
		pending = append(pending, n)
	}
	for len(pending) > 0 {
		n := pending[0]
		pending = pending[1:]
		for _, c := range graph[n] {
			if !result[c] {
				result[c] = true
				pending = append(pending, c)
			}
		}
	}
	return result
}

func refType(ref string) (stem, kind, variant string, ok bool) {
	r := strings.ToLower(ref)
	// First split numerical LOD/costume suffix, to support fooDiffuse.0 / fooLightMap.1.
	if idx := strings.LastIndex(r, "."); idx >= 0 {
		tail := r[idx+1:]
		if len(tail) > 0 && strings.IndexFunc(tail, func(c rune) bool { return c < '0' || c > '9' }) < 0 {
			variant = ref[idx:]
			ref = ref[:idx]
			r = r[:idx]
		}
	}
	for _, k := range []string{"normalmap", "diffuse", "lightmap", "metalmap", "shadowramp"} {
		if strings.HasSuffix(r, k) {
			return strings.ToLower(ref[:len(ref)-len(k)]), k, strings.ToLower(variant), true
		}
	}
	return "", "", "", false
}

// Classify a pair/triple only after confirming definitions and physical files.
func classifyBindings(path string, group []binding, resources map[string]string) (string, string) {
	if matched, reason := classifyFourSlotNNFix(path, group, resources); matched {
		return "NNFix", ""
	} else if reason != "" {
		return "", reason
	}
	slots := map[int]binding{}
	for _, b := range group {
		if b.slot < 3 {
			slots[b.slot] = b
		}
	}
	b0, ok0 := slots[0]
	b1, ok1 := slots[1]
	b2, ok2 := slots[2]
	if !ok0 || !ok1 {
		return "", "missing slots 0/1"
	}
	st0, k0, v0, s0 := refType(b0.ref)
	st1, k1, v1, s1 := refType(b1.ref)
	if !s0 || !s1 || st0 != st1 || v0 != v1 {
		return "", "texture stem/variant mismatch"
	}
	if !fileForRef(path, b0.ref, resources) || !fileForRef(path, b1.ref, resources) {
		return "", "missing DDS resources"
	}
	if k0 == "diffuse" && k1 == "lightmap" && !ok2 {
		return "NNFix", ""
	}
	if k0 == "normalmap" && k1 == "diffuse" && ok2 {
		st2, k2, v2, s2 := refType(b2.ref)
		if s2 && k2 == "lightmap" && st2 == st0 && v2 == v0 && fileForRef(path, b2.ref, resources) {
			return "ORFix", ""
		}
	}
	return "", "unsupported map layout"
}

// Enhanced entries are collected within a single branch, so a binding from one
// branch can never be mistaken for its opposite branch's texture.
func transformEnhanced(path string, content []byte, level string) ([]byte, []string, []string) {
	if v, c, ok := knownRecipe(path, content); ok {
		return v, c, nil
	}
	if isKnownPatchedCitlali(path, content) {
		return nil, nil, nil
	}
	// The generic enhanced classifier below handles verified four-slot
	// Diffuse/LightMap/MetalMap/ShadowRamp materials. No character name, hash,
	// or first-index signature is hard-coded here.
	out, baseChanges, baseManual := transformINI(path, content)
	if len(out) == 0 {
		out = content
	}
	if bytes.IndexByte(out, 0) >= 0 || isLibraryNamespace(out) || hasLegacyReflection(out) {
		if bytes.Equal(out, content) {
			return nil, nil, baseManual
		}
		return out, baseChanges, baseManual
	}
	if strings.EqualFold(filepath.Base(path), "ORFix.ini") {
		return nil, nil, nil
	}
	lines := strings.Split(string(out), "\n")
	ss := parseSections(lines)
	resources := resourceFiles(lines, ss)
	reachable := linkedSections(path, lines, ss, resources)
	hasTexFx := bytes.Contains(bytes.ToLower(out), []byte(`commandlist\texfx\`))
	after := map[int][]string{}
	var changes, manual []string
	// Groups are emitted right before a draw/run/branch/section end.
	for _, s := range ss {
		if !enhancedCandidate(s.name, reachable) {
			continue
		}
		var bindings []binding
		var sinceFix bool
		var haveOther69 bool
		lastMaskLine := -1
		var alreadyInserted bool
		flush := func(boundary int) {
			if len(bindings) == 0 {
				return
			}
			kind, reason := classifyBindings(path, bindings, resources)
			if kind == "" {
				if level == "enhanced" && reason != "missing slots 0/1" {
					manual = append(manual, fmt.Sprintf("%s [%d]: %s", s.name, boundary+1, reason))
				}
				bindings = nil
				haveOther69 = false
				lastMaskLine = -1
				alreadyInserted = false
				sinceFix = false
				return
			}
			if sinceFix || alreadyInserted {
				bindings = nil
				haveOther69 = false
				lastMaskLine = -1
				alreadyInserted = false
				sinceFix = false
				return
			}
			// Some sections have early command runs (calling a sublist) before binding.
			// Apply immediately before the next draw or material-changing command.
			// The no-ps-t69 reset is needed only in TexFx-enabled branch lists,
			// where earlier body mask can bleed into head/dress.
			indent := ""
			if len(bindings) > 0 {
				raw := normLine(lines[bindings[0].line])
				indent = raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]
			}
			inserts := []string{}
			// Strongly limit automatic resets to conditional CommandLists: in simple overrides
			// TexFx itself manages the state and global resets may be harmful.
			if strings.HasPrefix(strings.ToLower(s.name), "commandlist") && hasTexFx && !haveOther69 && level == "enhanced" {
				inserts = append(inserts, indent+"ps-t69 = null")
			}
			if kind == "NNFix" {
				inserts = append(inserts, indent+nnfixRun)
			} else {
				inserts = append(inserts, indent+orfixRun)
			}
			anchor := bindings[len(bindings)-1].line
			if lastMaskLine > anchor {
				anchor = lastMaskLine
			}
			after[anchor] = append(after[anchor], inserts...)
			changes = append(changes, fmt.Sprintf("%s [%s] 分支内贴图已匹配，绘制前插入 %s", s.name, kind, kind))
			bindings = nil
			haveOther69 = false
			lastMaskLine = -1
			alreadyInserted = true
			sinceFix = false
		}
		for i := s.start + 1; i < s.end; i++ {
			val := normLine(lines[i])
			norm := strings.TrimSpace(val)
			if norm == "" || strings.HasPrefix(norm, ";") {
				continue
			}
			if extBranch.MatchString(norm) {
				flush(i)
				alreadyInserted = false
				continue
			}
			if extFix.MatchString(norm) {
				sinceFix = true
				continue
			}
			if extTex69.MatchString(norm) {
				haveOther69 = true
				lastMaskLine = i
				continue
			}
			if m := extResource.FindStringSubmatch(val); m != nil {
				slot := 0
				fmt.Sscanf(m[1], "%d", &slot)
				if slot != 69 {
					// Keep slots beyond t3 too: a fifth slot must prevent an unsafe
					// four-slot classification rather than being silently ignored.
					// A new t0 begins a fresh material binding after an earlier completed set.
					if slot == 0 && len(bindings) > 0 {
						flush(i)
						alreadyInserted = false
					}
					bindings = append(bindings, binding{line: i, slot: slot, ref: m[2]})
					// Existing ORFix after previous draw doesn't count for next binding group.
					sinceFix = false
				}
				continue
			}
			if extAnyRun.MatchString(val) || extDraw.MatchString(val) {
				flush(i)
				// A draw without new material bindings must not trigger duplicate fixes.
				continue
			}
		}
		flush(s.end)
	}
	if len(after) == 0 {
		if bytes.Equal(out, content) {
			return nil, nil, append(baseManual, manual...)
		}
		return out, baseChanges, append(baseManual, manual...)
	}
	suffix := ""
	if bytes.Contains(out, []byte("\r\n")) {
		suffix = "\r"
	}
	built := make([]string, 0, len(lines)+len(changes)*2)
	for i, line := range lines {
		built = append(built, line)
		for _, v := range after[i] {
			built = append(built, v+suffix)
		}
	}
	final := []byte(strings.Join(built, "\n"))
	if bytes.Equal(final, content) {
		return nil, nil, append(baseManual, manual...)
	}
	filtered := []string{}
	for _, m := range baseManual {
		resolvedOrExplained := false
		for _, c := range changes {
			i := strings.Index(c, " [")
			if i >= 0 && strings.HasPrefix(m, c[:i]+"：") {
				resolvedOrExplained = true
				break
			}
		}
		if !resolvedOrExplained {
			for _, detail := range manual {
				i := strings.Index(detail, " [")
				if i >= 0 && strings.HasPrefix(m, detail[:i]+"：") {
					resolvedOrExplained = true
					break
				}
			}
		}
		if !resolvedOrExplained {
			filtered = append(filtered, m)
		}
	}
	return final, append(baseChanges, changes...), append(filtered, manual...)
}
