package engine

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func createINI(t *testing.T, content string, files ...string) string {
	t.Helper()
	d := t.TempDir()
	p := filepath.Join(d, "merged.ini")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(d, f), []byte("mock texture"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

const cmdListDemo = `[TextureOverrideDemoHead]
hash = deadbeef
match_first_index = 0
run = CommandListDemoHead
[TextureOverrideDemoBody]
hash = deadbeef
match_first_index = 1
run = CommandListDemoBody
[CommandListDemoHead]
if $swap == 0
 ps-t0 = ResourceDemoHeadDiffuse.0
 ps-t1 = ResourceDemoHeadLightMap.0
else
 ps-t0 = ResourceDemoHeadDiffuse.1
 ps-t1 = ResourceDemoHeadLightMap.1
endif
[CommandListDemoBody]
 ps-t0 = ResourceDemoBodyDiffuse
 ps-t1 = ResourceDemoBodyLightMap
 drawindexed = 99, 0, 0
 drawindexed = 12, 99, 0
[ResourceDemoHeadDiffuse.0]
filename = h0.dds
[ResourceDemoHeadLightMap.0]
filename = h0light.dds
[ResourceDemoHeadDiffuse.1]
filename = h1.dds
[ResourceDemoHeadLightMap.1]
filename = h1light.dds
[ResourceDemoBodyDiffuse]
filename = body.dds
[ResourceDemoBodyLightMap]
filename = bodylight.dds
`

func TestEnhancedBranchAndManualDraw(t *testing.T) {
	path := createINI(t, cmdListDemo, "h0.dds", "h0light.dds", "h1.dds", "h1light.dds", "body.dds", "bodylight.dds")
	patched, changes, _ := transformEnhanced(path, []byte(cmdListDemo), "enhanced")
	if len(changes) != 3 {
		t.Fatalf("expected three changes, got %v", changes)
	}
	if bytes.Count(patched, []byte(nnfixRun)) != 3 {
		t.Fatal("not inserted one NNFix per branch")
	}
	if !strings.Contains(string(patched), "run = "+strings.TrimPrefix(nnfixRun, "run = ")+"\n drawindexed = 99") {
		t.Fatal("manual draw is not prefaced")
	}
	p2, c2, _ := transformEnhanced(path, patched, "enhanced")
	if len(p2) > 0 || len(c2) > 0 {
		t.Fatalf("rerun is not idempotent: %v", c2)
	}
}
func TestEnhancedWeaponHeadOnlySkipped(t *testing.T) {
	ini := `[TextureOverrideWeaponHead]
hash = deadbeef
match_first_index = 0
ib = ResourceWeaponHeadIB
ps-t0 = ResourceWeaponHeadDiffuse
ps-t1 = ResourceWeaponHeadLightMap
[ResourceWeaponHeadDiffuse]
filename = a.dds
[ResourceWeaponHeadLightMap]
filename = b.dds
`
	path := createINI(t, ini, "a.dds", "b.dds")
	p, c, _ := transformEnhanced(path, []byte(ini), "enhanced")
	if len(p) > 0 || len(c) > 0 {
		t.Fatal("standalone weapon incorrectly modified")
	}
}
func TestEnhancedTexFxMaskRespected(t *testing.T) {
	ini := `[TextureOverrideAHead]
hash = deadbeef
match_first_index = 0
run = CommandListAHead
[TextureOverrideABody]
hash = deadbeef
match_first_index = 123
run = CommandListABody
[CommandListAHead]
ps-t0 = ResourceAHeadDiffuse
ps-t1 = ResourceAHeadLightMap
[CommandListABody]
ps-t0 = ResourceABodyDiffuse
ps-t1 = ResourceABodyLightMap
ps-t69 = ResourceABodyMask
run = CommandList\TexFx\Transparency.0
[ResourceAHeadDiffuse]
filename = a.dds
[ResourceAHeadLightMap]
filename = al.dds
[ResourceABodyDiffuse]
filename = b.dds
[ResourceABodyLightMap]
filename = bl.dds
[ResourceABodyMask]
filename = bm.dds
`
	path := createINI(t, ini, "a.dds", "al.dds", "b.dds", "bl.dds", "bm.dds")
	v, _, _ := transformEnhanced(path, []byte(ini), "enhanced")
	s := string(v)
	if !strings.Contains(s, "ps-t1 = ResourceAHeadLightMap\nps-t69 = null\n"+nnfixRun) {
		t.Fatal("head leak not reset")
	}
	if !strings.Contains(s, "ps-t69 = ResourceABodyMask\n"+nnfixRun+"\nrun = CommandList\\TexFx") {
		t.Fatal("body mask or fix ordering changed")
	}
}
func TestExperimentalTexFxExplicitOnly(t *testing.T) {
	ini := `[TextureOverrideDemoHead]
hash = abcdef01
match_first_index = 0
[TextureOverrideDemoBody]
hash = abcdef01
match_first_index = 1
ps-t69 = ResourceDemoBodyTexFX
run = CommandList\TexFx\Transparency.1
; Body
 drawindexed = 100, 0, 0
; Clothes_Chest_Mesh
 drawindexed = 120, 100, 0
; Skirt
 drawindexed = 200, 220, 0
; Arm_Ring
 drawindexed = 30, 420, 0
[ResourceDemoBodyTexFX]
filename = body.dds
`
	path := createINI(t, ini, "body.dds")
	patched, changes, manual := transformExperimentalTexFx(path, []byte(ini))
	if len(changes) != 1 || len(manual) != 0 {
		t.Fatalf("unexpected changes/manual %v %v", changes, manual)
	}
	s := string(patched)
	if strings.Count(s, `run = CommandList\TexFx\Component`) != 2 {
		t.Fatal("did not convert exactly mesh and ring")
	}
	if !strings.Contains(s, "drawindexed = 200, 220, 0") {
		t.Fatal("ordinary skirt draw altered")
	}
	if strings.Contains(s, "run = CommandList\\TexFx\\Transparency.1") {
		t.Fatal("global transparency not replaced")
	}
}
func TestRollbackProtectsManualEdits(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "test.ini")
	orig := []byte("original\n")
	if err := os.WriteFile(p, orig, 0644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.Local)
	curr := []byte("fixed\n")
	pl := plan{path: p, original: orig, patched: curr}
	pl.digest = [32]byte{} // fail backup if digest missing
	if _, err := backupAndReplace(pl, now); err == nil {
		t.Fatal("missing digest should fail")
	}
	pl.digest = sha256sum(orig)
	backup, err := backupAndReplace(pl, now)
	if err != nil {
		t.Fatal(err)
	}
	items := []rollbackEntry{{Path: "test.ini", Backup: filepath.Base(backup), OriginalSHA256: hexSha(orig), PatchedSHA256: hexSha(curr)}}
	if _, err := writeRollback(d, now, items); err != nil {
		t.Fatal(err)
	}
	// A manual edit must not be overwritten by 1-click rollback.
	if err := os.WriteFile(p, []byte("manually edited"), 0644); err != nil {
		t.Fatal(err)
	}
	n, _ := restoreLatest(d, now.Add(time.Second))
	if n != 0 {
		t.Fatal("manual edit overwritten")
	}
	if got, _ := os.ReadFile(p); string(got) != "manually edited" {
		t.Fatal("unexpected content after rollback")
	}
}
func sha256sum(b []byte) [32]byte { return sha256.Sum256(b) }
func TestEnhancedNamespacedModButNotLibrary(t *testing.T) {
	mod := "namespace = FischlMaidCustom\n" + rootINI
	p := fixture(t, mod, "XHeadNormalMap.dds", "XHeadDiffuse.dds", "XHeadLightMap.dds")
	// Note fixture uses rootINI's resources; unlike global libs it can be repaired.
	_, _, _ = transformEnhanced(p, []byte(mod), "enhanced")
	if isLibraryNamespace([]byte(mod)) {
		t.Fatal("namespaced mod classified as global library")
	}
	if !isLibraryNamespace([]byte("namespace = global\\ORFix\n")) {
		t.Fatal("global library not recognized")
	}
}

func TestIneffaIndependentEyeHeadRecognized(t *testing.T) {
	ini := `[TextureOverrideIneffaHead]
hash = 11111111
match_first_index = 0
ib = ResourceIneffaHeadIB
ps-t0 = ResourceIneffaHeadNormalMap
ps-t1 = ResourceIneffaHeadDiffuse
ps-t2 = ResourceIneffaHeadLightMap
run = CommandList\global\ORFix\ORFix
[TextureOverrideIneffaBody]
hash = 11111111
match_first_index = 1000
ib = ResourceIneffaBodyIB
ps-t0 = ResourceIneffaBodyNormalMap
ps-t1 = ResourceIneffaBodyDiffuse
ps-t2 = ResourceIneffaBodyLightMap
run = CommandList\global\ORFix\ORFix
[TextureOverrideIneffaEyeHead]
hash = 22222222
match_first_index = 0
ib = ResourceIneffaEyeHeadIB
ps-t0 = ResourceIneffaEyeHeadDiffuse
ps-t1 = ResourceIneffaEyeHeadLightMap
[TextureOverrideIneffaHairHead]
hash = 33333333
match_first_index = 0
ib = ResourceIneffaHairHeadIB
ps-t0 = ResourceIneffaHairHeadNormalMap
ps-t1 = ResourceIneffaHairHeadDiffuse
ps-t2 = ResourceIneffaHairHeadLightMap
run = CommandList\global\ORFix\ORFix
[ResourceIneffaHeadIB]
filename = IneffaHead.ib
[ResourceIneffaBodyIB]
filename = IneffaBody.ib
[ResourceIneffaEyeHeadIB]
filename = IneffaEyeHead.ib
[ResourceIneffaHairHeadIB]
filename = IneffaHairHead.ib
[ResourceIneffaHeadNormalMap]
filename = IneffaHeadNormalMap.dds
[ResourceIneffaHeadDiffuse]
filename = IneffaHeadDiffuse.dds
[ResourceIneffaHeadLightMap]
filename = IneffaHeadLightMap.dds
[ResourceIneffaBodyNormalMap]
filename = IneffaBodyNormalMap.dds
[ResourceIneffaBodyDiffuse]
filename = IneffaBodyDiffuse.dds
[ResourceIneffaBodyLightMap]
filename = IneffaBodyLightMap.dds
[ResourceIneffaEyeHeadDiffuse]
filename = IneffaEyeHeadDiffuse.dds
[ResourceIneffaEyeHeadLightMap]
filename = IneffaEyeHeadLightMap.dds
[ResourceIneffaHairHeadNormalMap]
filename = IneffaHairHeadNormalMap.dds
[ResourceIneffaHairHeadDiffuse]
filename = IneffaHairHeadDiffuse.dds
[ResourceIneffaHairHeadLightMap]
filename = IneffaHairHeadLightMap.dds
`
	files := []string{"IneffaHead.ib", "IneffaBody.ib", "IneffaEyeHead.ib", "IneffaHairHead.ib", "IneffaHeadNormalMap.dds", "IneffaHeadDiffuse.dds", "IneffaHeadLightMap.dds", "IneffaBodyNormalMap.dds", "IneffaBodyDiffuse.dds", "IneffaBodyLightMap.dds", "IneffaEyeHeadDiffuse.dds", "IneffaEyeHeadLightMap.dds", "IneffaHairHeadNormalMap.dds", "IneffaHairHeadDiffuse.dds", "IneffaHairHeadLightMap.dds"}
	path := createINI(t, ini, files...)
	patched, changes, manual := transformEnhanced(path, []byte(ini), "enhanced")
	if len(changes) != 1 || len(manual) != 0 {
		t.Fatalf("expected only eye submesh fix; changes=%v manual=%v", changes, manual)
	}
	if strings.Count(string(patched), nnfixRun) != 1 {
		t.Fatalf("expected one NNFix, got %d", strings.Count(string(patched), nnfixRun))
	}
	if !strings.Contains(string(patched), "[TextureOverrideIneffaEyeHead]\nhash = 22222222\nmatch_first_index = 0\nib = ResourceIneffaEyeHeadIB\nps-t0 = ResourceIneffaEyeHeadDiffuse\nps-t1 = ResourceIneffaEyeHeadLightMap\n"+nnfixRun) {
		t.Fatal("NNFix not inserted in EyeHead")
	}
	if strings.Count(string(patched), orfixRun) != 3 {
		t.Fatal("existing Head/Body/Hair ORFix calls changed")
	}
	p2, c2, _ := transformEnhanced(path, patched, "enhanced")
	if len(p2) > 0 || len(c2) > 0 {
		t.Fatalf("not idempotent: %v", c2)
	}
}

func TestEyeHeadWithoutVerifiedIBSkipped(t *testing.T) {
	ini := "[TextureOverrideIneffaHead]\nhash = 1\nmatch_first_index = 0\n[TextureOverrideIneffaBody]\nhash = 1\nmatch_first_index = 10\n[TextureOverrideIneffaEyeHead]\nhash = 2\nmatch_first_index = 0\nps-t0 = ResourceIneffaEyeHeadDiffuse\nps-t1 = ResourceIneffaEyeHeadLightMap\n[ResourceIneffaEyeHeadDiffuse]\nfilename = a.dds\n[ResourceIneffaEyeHeadLightMap]\nfilename = b.dds\n"
	path := createINI(t, ini, "a.dds", "b.dds")
	patched, changes, manual := transformEnhanced(path, []byte(ini), "enhanced")
	if len(patched) > 0 || len(changes) > 0 || len(manual) == 0 {
		t.Fatalf("unverified EyeHead must be skipped: %v %v", changes, manual)
	}
}

func TestIneffaEyeDiagnostic(t *testing.T) {
	ini := "[TextureOverrideIneffaHead]\nhash = 1\nmatch_first_index = 0\n[TextureOverrideIneffaBody]\nhash = 1\nmatch_first_index = 10\n[TextureOverrideIneffaEyeHead]\nhash = 2\nmatch_first_index = 0\nib = ResourceIneffaEyeHeadIB\nps-t0 = ResourceIneffaEyeHeadDiffuse\nps-t1 = ResourceIneffaEyeHeadLightMap\n[ResourceIneffaEyeHeadIB]\nfilename = Eye.ib\n[ResourceIneffaEyeHeadDiffuse]\nfilename = a.dds\n[ResourceIneffaEyeHeadLightMap]\nfilename = b.dds\n"
	path := createINI(t, ini, "Eye.ib", "a.dds", "b.dds")
	findings := diagnoseINI(path, []byte(ini))
	joined := strings.Join(findings, "\n")
	if !strings.Contains(joined, "独立眼睛网格") || !strings.Contains(joined, "缺少 NNFix") {
		t.Fatalf("diagnostic missed EyeHead: %v", findings)
	}
}

func TestRaidenShogunNotMisclassifiedAsGun(t *testing.T) {
	if !validSectionName("TextureOverrideRaidenShogunHead") {
		t.Fatal("RaidenShogunHead was rejected because Shogun contains the substring 'gun'")
	}
	if validSectionName("TextureOverrideWeaponHead") || validSectionName("TextureOverrideGunHead") {
		t.Fatal("weapon/gun mesh should remain excluded")
	}
}

func TestRaidenShogunThreeMeshesNNFix(t *testing.T) {
	ini := `[TextureOverrideRaidenShogunHead]
hash = 7a583c12
match_first_index = 0
ib = ResourceRaidenShogunHeadIB
ps-t0 = ResourceRaidenShogunHeadDiffuse
ps-t1 = ResourceRaidenShogunHeadLightMap
[TextureOverrideRaidenShogunBody]
hash = 7a583c12
match_first_index = 17769
ib = ResourceRaidenShogunBodyIB
ps-t0 = ResourceRaidenShogunBodyDiffuse
ps-t1 = ResourceRaidenShogunBodyLightMap
[TextureOverrideRaidenShogunDress]
hash = 7a583c12
match_first_index = 52473
ib = ResourceRaidenShogunDressIB
ps-t0 = ResourceRaidenShogunDressDiffuse
ps-t1 = ResourceRaidenShogunDressLightMap
[ResourceRaidenShogunHeadDiffuse]
filename = HeadDiffuse.dds
[ResourceRaidenShogunHeadLightMap]
filename = HeadLightMap.dds
[ResourceRaidenShogunBodyDiffuse]
filename = BodyDiffuse.dds
[ResourceRaidenShogunBodyLightMap]
filename = BodyLightMap.dds
[ResourceRaidenShogunDressDiffuse]
filename = DressDiffuse.dds
[ResourceRaidenShogunDressLightMap]
filename = DressLightMap.dds
`
	path := createINI(t, ini, "HeadDiffuse.dds", "HeadLightMap.dds", "BodyDiffuse.dds", "BodyLightMap.dds", "DressDiffuse.dds", "DressLightMap.dds")
	patched, changes, manual := transformEnhanced(path, []byte(ini), "enhanced")
	if len(changes) != 3 || len(manual) != 0 || strings.Count(string(patched), nnfixRun) != 3 {
		t.Fatalf("expected three Raiden NNFix calls: changes=%v manual=%v", changes, manual)
	}
}

func TestFindORFixInParentModsFolder(t *testing.T) {
	base := t.TempDir()
	mods := filepath.Join(base, "Mods")
	mod := filepath.Join(mods, "CharacterA")
	if err := os.MkdirAll(filepath.Join(mods, "global"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(mod, 0755); err != nil {
		t.Fatal(err)
	}
	lib := []byte("namespace = global\\ORFix\n[CommandList\\global\\ORFix\\NNFix]\n")
	if err := os.WriteFile(filepath.Join(mods, "global", "ORFix.ini"), lib, 0644); err != nil {
		t.Fatal(err)
	}
	if !findLibraryInAncestors(mod, true) {
		t.Fatal("did not find ORFix in parent Mods folder")
	}
}
