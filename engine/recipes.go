package engine

import (
	"crypto/sha256"
	"embed"
	"os"
	"path/filepath"
	"strings"
)

// Only a byte-for-byte match of a previously game-verified user-supplied INI
// can trigger this mesh-specific transparency recipe. No heuristic TexFx edits.
//
//go:embed known_fixes/Citlali_high_priest_verified_v2.ini
var recipeFiles embed.FS

var knownCitlaliOriginal = [32]byte{0x2d, 0xf4, 0x45, 0x3b, 0x1c, 0xa0, 0xf1, 0x53, 0x7d, 0x6c, 0x9a, 0x03, 0x0b, 0xd9, 0x4f, 0x99, 0xad, 0xf5, 0x42, 0x5b, 0x4c, 0xc2, 0x74, 0x44, 0x4b, 0x0e, 0xb4, 0x22, 0x7c, 0x01, 0x9d, 0x06}
var knownCitlaliFailedV1 = [32]byte{0x8b, 0x4f, 0x56, 0x8f, 0x4a, 0xac, 0x0f, 0x4d, 0xb3, 0xfb, 0xeb, 0x25, 0x34, 0x08, 0xa7, 0x92, 0x51, 0x95, 0xb6, 0x54, 0x26, 0xb5, 0x72, 0x1f, 0x3d, 0xb8, 0x33, 0xee, 0xd0, 0x41, 0x14, 0x0f}

func knownRecipe(path string, content []byte) ([]byte, []string, bool) {
	if !strings.EqualFold(filepath.Base(path), "Citlali.ini") {
		return nil, nil, false
	}
	h := sha256.Sum256(content)
	if h != knownCitlaliOriginal && h != knownCitlaliFailedV1 {
		return nil, nil, false
	}
	// Avoid applying a known-good INI to a partially copied mod without the mesh and mask assets.
	for _, name := range []string{"CitlaliBody.ib", "CitlaliBodyTexfx.dds", "CitlaliBodyDiffuse.dds", "CitlaliBodyLightMap.dds"} {
		info, err := os.Stat(filepath.Join(filepath.Dir(path), name))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return nil, nil, false
		}
	}
	patched, err := recipeFiles.ReadFile("known_fixes/Citlali_high_priest_verified_v2.ini")
	if err != nil {
		return nil, nil, false
	}
	return patched, []string{"[TexFx 专项] 茜特菈莉 High Priest：胸腹网纱和戒指采用已验证的网格级透明绘制；保留全部原资源"}, true
}

func isKnownPatchedCitlali(path string, content []byte) bool {
	if !strings.EqualFold(filepath.Base(path), "Citlali.ini") {
		return false
	}
	return sha256.Sum256(content) == [32]byte{0x50, 0x43, 0x67, 0x07, 0xbf, 0xea, 0xb6, 0x18, 0xa9, 0xb4, 0xd6, 0x47, 0x17, 0xd9, 0x54, 0x0d, 0x03, 0x04, 0x77, 0xf6, 0xe2, 0x37, 0x63, 0x70, 0x66, 0x54, 0x14, 0x75, 0x82, 0xa8, 0x7b, 0x99}
}
