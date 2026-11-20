package engine

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func makeLegacyNingguangFixture(t *testing.T) (string, string, []byte, []byte) {
	t.Helper()
	root := t.TempDir()
	ini := filepath.Join(root, "Ningguang.ini")
	config := `; Ningguang synthetic regression test
[TextureOverrideNingguangPosition]
hash = 55b43e99
vb0 = ResourceNingguangPosition
[TextureOverrideNingguangBlend]
hash = 9f7dc19c
vb1 = ResourceNingguangBlend
[TextureOverrideNingguangTexcoord]
hash = 906ad233
vb1 = ResourceNingguangTexcoord
[TextureOverrideNingguangIB]
hash = abdc3768
[TextureOverrideNingguangHead]
hash = abdc3768
ib = ResourceNingguangHeadIB
ps-t0 = ResourceNingguangHeadDiffuse
ps-t1 = ResourceNingguangHeadLightMap
[TextureOverrideNingguangBody]
hash = abdc3768
ib = ResourceNingguangBodyIB
ps-t0 = ResourceNingguangBodyDiffuse
ps-t1 = ResourceNingguangBodyLightMap
[TextureOverrideNingguangDress]
hash = abdc3768
ib = ResourceNingguangDressIB
ps-t0 = ResourceNingguangDressDiffuse
ps-t1 = ResourceNingguangDressLightMap
[TextureOverride41FixVertexLimitRaise0]
hash = 6d197625
[ResourceNingguangBlend]
type = Buffer
stride = 32
filename = NingguangBlend.buf
`
	for _, part := range []string{"Head", "Body", "Dress"} {
		for _, suffix := range []string{"Diffuse", "LightMap"} {
			name := "Ningguang" + part + suffix + ".dds"
			config += "[ResourceNingguang" + part + suffix + "]\nfilename = " + name + "\n"
			if err := os.WriteFile(filepath.Join(root, name), []byte{1, 2, 3}, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	oldINI := []byte(strings.ReplaceAll(config, "\n", "\r\n"))
	blend := make([]byte, 32*3)
	for row := 0; row < 3; row++ {
		offset := row * 32
		binary.LittleEndian.PutUint32(blend[offset:], math.Float32bits(1))
		binary.LittleEndian.PutUint32(blend[offset+16:], uint32(row+20))
	}
	if err := os.WriteFile(ini, oldINI, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "NingguangBlend.buf"), blend, 0644); err != nil {
		t.Fatal(err)
	}
	return root, ini, oldINI, blend
}

func TestV32NingguangMigrationAndRollback(t *testing.T) {
	root, ini, oldINI, oldBlend := makeLegacyNingguangFixture(t)
	r := scan(root, "enhanced")
	if len(r.repairs) != 1 {
		t.Fatalf("need one multi-file plan, got %d, manual: %v", len(r.repairs), r.manual)
	}
	p := r.repairs[0]
	if len(p.companions) != 1 {
		t.Fatalf("missing companion BUF: %+v", p.changes)
	}
	if bytes.Count(p.patched, []byte(nnfixRun)) != 3 {
		t.Fatal("expected 3 NNFix calls")
	}
	if activeHashes(p.patched)["ad75352c"] != 4 {
		t.Fatal("IB migration missing")
	}
	if len(ningguangRemap) != 119 {
		t.Fatal("bad remap table")
	}
	t0 := time.Now()
	entries, backs, err := applyGroup(p, t0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || len(backs) != 2 {
		t.Fatalf("want 2 backups and entries; got %d %d", len(entries), len(backs))
	}
	for i := range entries {
		entries[i].Path = rel(root, entries[i].Path)
		entries[i].Backup = rel(root, entries[i].Backup)
	}
	if _, err := writeRollback(root, t0, entries); err != nil {
		t.Fatal(err)
	}
	if rerun := scan(root, "enhanced"); len(rerun.repairs) != 0 {
		t.Fatalf("second scan must be idempotent: %+v", rerun.repairs)
	}
	restored, logs := restoreLatest(root, t0.Add(time.Second))
	if restored != 2 {
		t.Fatalf("expected two restored files got %d: %v", restored, logs)
	}
	a, _ := os.ReadFile(ini)
	b, _ := os.ReadFile(filepath.Join(root, "NingguangBlend.buf"))
	if !bytes.Equal(a, oldINI) || !bytes.Equal(b, oldBlend) {
		t.Fatal("rollback bytes differ from originals")
	}
}

func TestV32NingguangIncompleteSignatureNeverRemaps(t *testing.T) {
	_, ini, orig, _ := makeLegacyNingguangFixture(t)
	changed := bytes.Replace(orig, []byte("hash = 906ad233"), []byte("hash = 12345678"), 1)
	newer, bp, ch, manual := migrateNingguangLegacy(ini, changed)
	if newer != nil || bp != nil || len(ch) != 0 || len(manual) == 0 {
		t.Fatalf("unexpected partial upgrade: changes=%v manual=%v", ch, manual)
	}
}

func TestV32NingguangInvalidBlendNeverRemaps(t *testing.T) {
	root, ini, orig, oldBlend := makeLegacyNingguangFixture(t)
	corrupt := bytes.Clone(oldBlend)
	binary.LittleEndian.PutUint32(corrupt[16:], 800)
	if err := os.WriteFile(filepath.Join(root, "NingguangBlend.buf"), corrupt, 0644); err != nil {
		t.Fatal(err)
	}
	newer, bp, _, manual := migrateNingguangLegacy(ini, orig)
	if newer != nil || bp != nil || len(manual) == 0 {
		t.Fatalf("invalid VG should abort all edits: %v", manual)
	}
}

func TestV32TexFxStandardPassWithoutChangingTransparency(t *testing.T) {
	root := t.TempDir()
	ini := filepath.Join(root, "RaidenShogun.ini")
	text := `[TextureOverrideRaidenShogunHead]
hash = aaaaaaaa
ib = ResourceHeadIB
ps-t0 = ResourceRaidenShogunHeadDiffuse
ps-t1 = ResourceRaidenShogunHeadLightMap
[TextureOverrideRaidenShogunBody]
hash = bbbbbbbb
ib = ResourceBodyIB
ps-t0 = ResourceRaidenShogunBodyDiffuse
ps-t1 = ResourceRaidenShogunBodyLightMap
[TextureOverrideRaidenShogunDress]
hash = cccccccc
ib = ResourceDressIB
ps-t0 = ResourceRaidenShogunDressDiffuse
ps-t1 = ResourceRaidenShogunDressLightMap
ps-t69 = ResourceRaidenShogunDressTransparencyMap
run = CommandList\TexFx\Transparency.0
`
	for _, part := range []string{"Head", "Body", "Dress"} {
		for _, suffix := range []string{"Diffuse", "LightMap"} {
			name := "RaidenShogun" + part + suffix + ".dds"
			text += "[ResourceRaidenShogun" + part + suffix + "]\nfilename = " + name + "\n"
			if err := os.WriteFile(filepath.Join(root, name), []byte{1}, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	text += "[ResourceRaidenShogunDressTransparencyMap]\nfilename = mask.dds\n"
	os.WriteFile(filepath.Join(root, "mask.dds"), []byte{1}, 0644)
	os.WriteFile(ini, []byte(text), 0644)
	got := scan(root, "enhanced")
	if len(got.repairs) != 1 {
		t.Fatalf("expected 1 plan, got %d manual %v", len(got.repairs), got.manual)
	}
	actual := string(got.repairs[0].patched)
	if strings.Count(actual, nnfixRun) != 3 {
		t.Fatalf("expected 3 NNFix calls\n%s", actual)
	}
	originalMask := "ps-t69 = ResourceRaidenShogunDressTransparencyMap"
	if !strings.Contains(actual, originalMask) || !strings.Contains(actual, `run = CommandList\TexFx\Transparency.0`) {
		t.Fatal("texfx changed")
	}
	if strings.Index(actual, nnfixRun) > strings.Index(actual, "[TextureOverrideRaidenShogunDress]") {
		// First NNFix belongs to head. Dress run must be placed between mask and TexFx.
	}
	dress := actual[strings.Index(actual, "[TextureOverrideRaidenShogunDress]"):]
	if !(strings.Index(dress, originalMask) < strings.Index(dress, nnfixRun) && strings.Index(dress, nnfixRun) < strings.Index(dress, `run = CommandList\TexFx\Transparency.0`)) {
		t.Fatal("wrong NNFix/TexFx order")
	}
	if got2 := scan(root, "enhanced"); len(got2.repairs) != 1 {
		t.Fatal("dry scan must not write")
	}
}
