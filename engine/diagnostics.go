package engine

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// isORFixLibrary and isTexFxLibrary are environment checks only; they never
// modify global libraries. The user may intentionally keep libraries outside
// the scanned Mods root, so missing-library findings are warnings, not errors.
func isORFixLibrary(name string, content []byte) bool {
	if strings.EqualFold(filepath.Base(name), "ORFix.ini") {
		return true
	}
	m := extNamespace.FindSubmatch(content)
	if m == nil {
		return false
	}
	ns := strings.ToLower(strings.TrimSpace(string(m[1])))
	return ns == `global\orfix` || ns == "orfix"
}

func isTexFxLibrary(name string, content []byte) bool {
	base := strings.ToLower(filepath.Base(name))
	if base == "main.ini" && strings.Contains(strings.ToLower(string(content)), "namespace = texfx") {
		return true
	}
	m := extNamespace.FindSubmatch(content)
	if m == nil {
		return false
	}
	ns := strings.ToLower(strings.TrimSpace(string(m[1])))
	return ns == "texfx" || ns == `global\texfx`
}

// diagnoseINI reports what the scanner can prove from INI bindings and the
// referenced files. It deliberately does not infer visual colors from channel
// names or claim that NNFix/ORFix alone will fix every green mesh.
func diagnoseINI(path string, content []byte) []string {
	if len(content) == 0 || strings.EqualFold(filepath.Base(path), "ORFix.ini") || isLibraryNamespace(content) {
		return nil
	}
	lines := strings.Split(string(content), "\n")
	ss := parseSections(lines)
	resources := resourceFiles(lines, ss)
	var out []string
	if hasLegacyReflection(content) {
		out = append(out, filepath.Base(path)+"：发现旧式 Reflection ShaderRegex/CommandList；需检查是否与 ORFix 冲突")
	}
	for _, sec := range ss {
		if !strings.HasPrefix(strings.ToLower(sec.name), "textureoverride") {
			continue
		}
		if !validSectionName(sec.name) || !isCharacterOutfitCandidate(sec.name, ss) {
			continue
		}
		if isCharacterSubcomponentHead(sec.name, ss) && !verifiedSubcomponentIB(path, lines, sec, resources) {
			out = append(out, sec.name+"：疑似独立眼睛/头发子网格，但 IB 文件未核实；自动修复会跳过")
			continue
		}
		var group []binding
		var hasNN, hasOR, hasTex69, hasTexFx, hasCustomTransparency bool
		for i := sec.start + 1; i < sec.end; i++ {
			v := normLine(lines[i])
			if m := reTex.FindStringSubmatch(v); m != nil && m[2] == "" {
				slot, err := strconv.Atoi(m[3])
				if err == nil && slot != 69 {
					group = append(group, binding{line: i, slot: slot, ref: m[4]})
				}
			}
			if m := reRunOR.FindStringSubmatch(v); m != nil {
				if strings.EqualFold(m[1], "NNFix") {
					hasNN = true
				} else {
					hasOR = true
				}
			}
			if extTex69.MatchString(v) {
				hasTex69 = true
			}
			if strings.Contains(strings.ToLower(v), `commandlist\texfx\`) {
				hasTexFx = true
			}
			if strings.Contains(strings.ToLower(v), "customshadertransparency") {
				hasCustomTransparency = true
			}
		}
		kind, reason := classifyBindings(path, group, resources)
		role := ""
		low := strings.ToLower(sec.name)
		if strings.Contains(low, "eyehead") || strings.Contains(low, "eyeshead") {
			role = "独立眼睛网格；"
		}
		if strings.Contains(low, "hairhead") {
			role = "独立头发网格；"
		}
		switch kind {
		case "NNFix":
			if isFourSlotNNFixGroup(path, group, resources) {
				if hasNN {
					out = append(out, sec.name+"：四槽位 Diffuse + LightMap + MetalMap + ShadowRamp，已存在 NNFix")
				} else {
					out = append(out, sec.name+"：四槽位 Diffuse + LightMap + MetalMap + ShadowRamp，缺少 NNFix；满足资源校验，可作为 NNFix 修复候选")
				}
			} else if hasNN {
				out = append(out, sec.name+"："+role+"双贴图 Diffuse + LightMap，已存在 NNFix；若仍发绿，需进一步检查 Shader、倒影或贴图实际语义")
			} else {
				out = append(out, sec.name+"："+role+"双贴图 Diffuse + LightMap，缺少 NNFix；满足资源校验，可作为 NNFix 修复候选")
			}
		case "ORFix":
			if hasOR {
				out = append(out, sec.name+"：三贴图 NormalMap + Diffuse + LightMap，已存在 ORFix")
			} else {
				out = append(out, sec.name+"：三贴图 NormalMap + Diffuse + LightMap，缺少 ORFix；满足资源校验，可作为 ORFix 修复候选")
			}
		default:
			if len(group) > 0 && reason != "missing slots 0/1" {
				out = append(out, sec.name+"：贴图槽位未能严格分类（"+reason+"）；不推断应使用 ORFix 还是 NNFix")
			}
		}
		if hasTex69 || hasTexFx || hasCustomTransparency {
			kind := "检测到透明相关配置"
			if hasTex69 && hasTexFx {
				kind = "检测到 ps-t69 + TexFx 透明遮罩"
			}
			if hasCustomTransparency {
				kind += " / CustomShaderTransparency"
			}
			out = append(out, sec.name+"："+kind+"；透明网格需按 IB/UV/绘制范围分析，不能仅靠全局 NNFix 推断")
		}
	}
	return out
}

// Check a few conventional parent locations because users often put the EXE
// inside one character's Mod folder while global libraries live in Mods/.
// This is a bounded filesystem check and never changes files.
func findLibraryInAncestors(root string, wantORFix bool) bool {
	current, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	for depth := 0; depth < 5; depth++ {
		candidates := []string{}
		if wantORFix {
			candidates = append(candidates,
				filepath.Join(current, "ORFix.ini"),
				filepath.Join(current, "global", "ORFix.ini"),
				filepath.Join(current, "global", "ORFix", "ORFix.ini"),
			)
		} else {
			candidates = append(candidates,
				filepath.Join(current, "TexFx", "Main.ini"),
				filepath.Join(current, "global", "TexFx", "Main.ini"),
				filepath.Join(current, "TexFx.ini"),
			)
		}
		for _, candidate := range candidates {
			b, e := os.ReadFile(candidate)
			if e != nil {
				continue
			}
			if wantORFix && isORFixLibrary(candidate, b) {
				return true
			}
			if !wantORFix && isTexFxLibrary(candidate, b) {
				return true
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return false
}
