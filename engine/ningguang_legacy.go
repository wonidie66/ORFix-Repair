package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Game-verified migration: historical Ningguang Base 4.3 --> 4.4+ hashes
// plus the exact 119-entry vertex-group permutation from LeoTools/Fix4.4.py.
// This mapping is NOT a universal 7.2 hash table and is only applied to this
// recognized old Ningguang export when the entire signature is present.
var ningguangRemap = []uint32{
	70, 6, 20, 21, 22, 12, 13, 14, 15, 16, 7, 17,
	18, 19, 8, 9, 10, 11, 4, 5, 69, 54, 53, 94, 111, 115, 71, 88,
	92, 91, 52, 114, 116, 117, 118, 95, 93, 67, 72, 65, 68, 60, 61,
	62, 63, 64, 66, 55, 56, 57, 58, 59, 40, 44, 45, 32, 23, 29, 24,
	33, 37, 38, 30, 89, 90, 73, 79, 76, 82, 85, 86, 74, 75, 77, 78,
	80, 81, 83, 84, 87, 2, 0, 112, 113, 96, 102, 99, 105, 108, 109,
	97, 98, 100, 101, 103, 104, 106, 107, 110, 41, 43, 48, 28, 3,
	1, 31, 36, 35, 34, 26, 27, 25, 39, 49, 51, 47, 46, 42, 50,
}

type hashPair struct {
	old, new string
	count    int
}

var ningguangHashPairs = []hashPair{
	{"55b43e99", "f9e1b52b", 1},
	{"9f7dc19c", "735eaea4", 1},
	{"906ad233", "1f0ab400", 1},
	{"abdc3768", "ad75352c", 4},
	{"6d197625", "4c2f9a0a", 1},
}
var reLegacyHashLine = regexp.MustCompile(`(?im)^([ \t]*hash[ \t]*=[ \t]*)([0-9a-f]{8})([ \t]*(?:;[^\r\n]*)?\r?)$`)

func remapNingguangBlend(src []byte) ([]byte, int, error) {
	if len(ningguangRemap) != 119 {
		return nil, 0, fmt.Errorf("顶点组映射表长度错误: %d", len(ningguangRemap))
	}
	if len(src) == 0 || len(src)%32 != 0 {
		return nil, 0, fmt.Errorf("Blend.buf 必须是非空的 32 字节记录，实际 %d", len(src))
	}
	dst := bytes.Clone(src)
	total := 0
	for i := 0; i < len(src); i += 32 {
		for k := 0; k < 4; k++ {
			weight := math.Float32frombits(binary.LittleEndian.Uint32(src[i+k*4:]))
			idxPos := i + 16 + k*4
			idx := binary.LittleEndian.Uint32(src[idxPos:])
			if math.IsNaN(float64(weight)) || math.IsInf(float64(weight), 0) || weight < 0 || weight > 1.001 {
				return nil, 0, fmt.Errorf("偏移 %d 存在异常权重", i+k*4)
			}
			if weight != 0 {
				if idx >= uint32(len(ningguangRemap)) {
					return nil, 0, fmt.Errorf("活动顶点组索引 %d 超出映射表", idx)
				}
				binary.LittleEndian.PutUint32(dst[idxPos:], ningguangRemap[idx])
				total++
			}
		}
	}
	if total == 0 {
		return nil, 0, errors.New("Blend.buf 未发现有效骨骼权重")
	}
	return dst, total, nil
}

func activeHashes(content []byte) map[string]int {
	result := map[string]int{}
	for _, m := range reLegacyHashLine.FindAllSubmatch(content, -1) {
		result[strings.ToLower(string(m[2]))]++
	}
	return result
}

// Returns new INI + companion BUF edit. Aborts without any partial change if
// signature, required files, remap indices, or material layout are uncertain.
func migrateNingguangLegacy(path string, content []byte) ([]byte, *plan, []string, []string) {
	if !strings.EqualFold(filepath.Base(path), "Ningguang.ini") {
		return nil, nil, nil, nil
	}
	hashes := activeHashes(content)
	oldCount, newCount := 0, 0
	for _, pair := range ningguangHashPairs {
		oldCount += hashes[pair.old]
		newCount += hashes[pair.new]
	}
	if oldCount == 0 {
		return nil, nil, nil, nil
	}
	if newCount > 0 {
		return nil, nil, nil, []string{"[凝光历史迁移] 新旧 Hash 混用；避免部分替换，未修改"}
	}
	for _, pair := range ningguangHashPairs {
		if hashes[pair.old] != pair.count {
			return nil, nil, nil, []string{fmt.Sprintf("[凝光历史迁移] 旧 Hash %s 应出现 %d 次，实际 %d；不满足已验证结构，未修改", pair.old, pair.count, hashes[pair.old])}
		}
	}
	lines := strings.Split(string(content), "\n")
	ss := parseSections(lines)
	refs := resourceFiles(lines, ss)
	for _, part := range []string{"Head", "Body", "Dress"} {
		name := "TextureOverrideNingguang" + part
		found := false
		for _, s := range ss {
			if strings.EqualFold(s.name, name) {
				found = true
				partBlock := strings.ToLower(strings.Join(lines[s.start+1:s.end], "\n"))
				if !strings.Contains(partBlock, "ps-t0 = resourceningguang"+strings.ToLower(part)+"diffuse") || !strings.Contains(partBlock, "ps-t1 = resourceningguang"+strings.ToLower(part)+"lightmap") {
					return nil, nil, nil, []string{"[凝光历史迁移] " + name + " 贴图布局未知，未修改"}
				}
				for _, suffix := range []string{"Diffuse", "LightMap"} {
					if !fileForRef(path, "ResourceNingguang"+part+suffix, refs) {
						return nil, nil, nil, []string{"[凝光历史迁移] " + part + " 贴图缺失，未修改"}
					}
				}
				break
			}
		}
		if !found {
			return nil, nil, nil, []string{"[凝光历史迁移] 找不到 " + name + "，未修改"}
		}
	}
	blendName, ok := refs["resourceningguangblend"]
	if !ok || strings.TrimSpace(blendName) == "" {
		return nil, nil, nil, []string{"[凝光历史迁移] 缺少 Blend 文件定义，未修改"}
	}
	blendName = strings.Trim(blendName, `"`)
	blendName = strings.ReplaceAll(blendName, `\`, string(os.PathSeparator))
	if !filepath.IsLocal(blendName) {
		return nil, nil, nil, []string{"[凝光历史迁移] Blend 路径不安全，未修改"}
	}
	blendPath := filepath.Join(filepath.Dir(path), blendName)
	blendBytes, err := os.ReadFile(blendPath)
	if err != nil {
		return nil, nil, nil, []string{"[凝光历史迁移] 读取 Blend 失败：" + err.Error()}
	}
	remapped, weightCount, err := remapNingguangBlend(blendBytes)
	if err != nil {
		return nil, nil, nil, []string{"[凝光历史迁移] " + err.Error()}
	}
	if bytes.Equal(remapped, blendBytes) {
		return nil, nil, nil, []string{"[凝光历史迁移] Blend 可能已重映射，拒绝重复处理"}
	}
	counts := map[string]int{}
	updated := reLegacyHashLine.ReplaceAllFunc(content, func(line []byte) []byte {
		m := reLegacyHashLine.FindSubmatch(line)
		if len(m) == 0 {
			return line
		}
		v := strings.ToLower(string(m[2]))
		for _, pair := range ningguangHashPairs {
			if v == pair.old {
				counts[v]++
				return []byte(string(m[1]) + pair.new + string(m[3]))
			}
		}
		return line
	})
	for _, pair := range ningguangHashPairs {
		if counts[pair.old] != pair.count {
			return nil, nil, nil, []string{"[凝光历史迁移] Hash 替换计数异常，未修改"}
		}
	}
	companion := &plan{path: blendPath, original: blendBytes, digest: sha256.Sum256(blendBytes), patched: remapped, changes: []string{fmt.Sprintf("NingguangBlend.buf 按官方 119 项映射迁移（%d 个非零权重索引）", weightCount)}}
	return updated, companion, []string{"[凝光历史迁移] 4.3→4.4+ 历史 Hash 五组已核实更新；Blend 119 项映射已应用；共 " + fmt.Sprint(weightCount) + " 个权重索引"}, nil
}
