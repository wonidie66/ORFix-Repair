package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T, ini string, filenames ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, file := range filenames {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("dds mock"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "merged.ini")
	if err := os.WriteFile(path, []byte(ini), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const rootINI = `[TextureOverrideXHead]
hash = deadbeef
match_first_index = 0
run = CommandListXHead
;ps-t0 = ResourceXHeadNormalMap
;ps-t1 = ResourceXHeadDiffuse
ps-t2 = ResourceXHeadLightMap

[ResourceXHeadNormalMap]
filename = XHeadNormalMap.dds
[ResourceXHeadDiffuse]
filename = XHeadDiffuse.dds
[ResourceXHeadLightMap]
filename = XHeadLightMap.dds
[TextureOverrideXBody]
hash = 01234567
`

func threePaths() []string {
	return []string{"XHeadNormalMap.dds", "XHeadDiffuse.dds", "XHeadLightMap.dds"}
}
func TestRestoresAndIdempotent(t *testing.T) {
	path := fixture(t, rootINI, threePaths()...)
	first, changes, manual := transformINI(path, []byte(rootINI))
	if len(changes) != 1 || len(manual) != 0 {
		t.Fatalf("changes=%v manual=%v", changes, manual)
	}
	for _, required := range []string{"ps-t0 = ResourceXHeadNormalMap", "ps-t1 = ResourceXHeadDiffuse", orfixRun} {
		if !strings.Contains(string(first), required) {
			t.Fatalf("missing %s", required)
		}
	}
	second, changes, manual := transformINI(path, first)
	if len(second) > 0 || len(changes) > 0 || len(manual) > 0 {
		t.Fatalf("second run modified file: %q %v %v", second, changes, manual)
	}
	r := scan(filepath.Dir(path))
	if len(r.repairs) != 1 {
		t.Fatalf("scan repairs=%d", len(r.repairs))
	}
	backup, err := backupAndReplace(r.repairs[0], time.Date(2026, 11, 17, 4, 45, 33, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(backup) != "merged.ini_2026-11-17_04-45-33.bak" {
		t.Fatal(backup)
	}
	original, err := os.ReadFile(backup)
	if err != nil || !bytes.Equal(original, []byte(rootINI)) {
		t.Fatal("backup mismatch", err)
	}
	if len(scan(filepath.Dir(path)).repairs) != 0 {
		t.Fatal("not idempotent")
	}
}
func TestNoResourceSkips(t *testing.T) {
	path := fixture(t, rootINI, "XHeadLightMap.dds")
	data, changes, manual := transformINI(path, []byte(rootINI))
	if len(data) != 0 || len(changes) != 0 || len(manual) == 0 {
		t.Fatalf("unexpected: %v %v %v", len(data), changes, manual)
	}
}
func TestLegacySkips(t *testing.T) {
	ini := rootINI + "\n[ShaderRegexCharReflection]\nshader_model = ps_5_0\n"
	path := fixture(t, ini, threePaths()...)
	data, changes, manual := transformINI(path, []byte(ini))
	if len(data) != 0 || len(changes) != 0 || len(manual) == 0 {
		t.Fatalf("legacy not skipped: %v %v %v", len(data), changes, manual)
	}
}
func TestConditionalSkips(t *testing.T) {
	ini := strings.Replace(rootINI, "run = CommandListXHead", "if $swapvar == 0\nrun = CommandListXHead\nendif", 1)
	path := fixture(t, ini, threePaths()...)
	data, changes, manual := transformINI(path, []byte(ini))
	if len(data) != 0 || len(changes) != 0 || len(manual) == 0 {
		t.Fatalf("condition not skipped: %v %v %v", len(data), changes, manual)
	}
}
func TestCRLFPreserved(t *testing.T) {
	ini := strings.ReplaceAll(rootINI, "\n", "\r\n")
	path := fixture(t, ini, threePaths()...)
	data, changes, _ := transformINI(path, []byte(ini))
	if len(changes) != 1 || len(data) == 0 {
		t.Fatal("no edit")
	}
	if bytes.Contains(bytes.ReplaceAll(data, []byte("\r\n"), nil), []byte("\n")) {
		t.Fatal("mixed line endings")
	}
}
func TestNNFixCandidateSkipped(t *testing.T) {
	ini := `[TextureOverrideXLynetteDress]
match_first_index = 2
ps-t0 = ResourceXDressDiffuse
ps-t1 = ResourceXDressLightMap
`
	path := fixture(t, ini)
	data, changes, manual := transformINI(path, []byte(ini))
	if len(data) != 0 || len(changes) != 0 || len(manual) == 0 {
		t.Fatalf("unsafe two-texture auto repair: %v %v %v", len(data), changes, manual)
	}
}
func TestActiveThreeSlotGetsORFix(t *testing.T) {
	ini := strings.ReplaceAll(rootINI, ";ps-t", "ps-t")
	path := fixture(t, ini, threePaths()...)
	data, changes, _ := transformINI(path, []byte(ini))
	if len(changes) != 1 || strings.Count(string(data), orfixRun) != 1 {
		t.Fatalf("changes=%v", changes)
	}
	if strings.Count(string(data), "ps-t0 =") != 1 {
		t.Fatal("duplicated t0")
	}
}
func TestGlobalAndDisabledSkipped(t *testing.T) {
	ini := rootINI + "\nnamespace = global\\ORFix\n"
	path := fixture(t, ini, threePaths()...)
	if data, _, _ := transformINI(path, []byte(ini)); len(data) != 0 {
		t.Fatal("global was modified")
	}
	disabled := filepath.Join(filepath.Dir(path), "DISABLED_character.ini")
	if err := os.WriteFile(disabled, []byte(rootINI), 0644); err != nil {
		t.Fatal(err)
	}
	r := scan(filepath.Dir(path))
	if len(r.repairs) > 0 {
		t.Fatal("global/disabled should not be patched")
	}
}
func TestSecondBackupNameCollision(t *testing.T) {
	path := fixture(t, rootINI, threePaths()...)
	r := scan(filepath.Dir(path))
	stamp := time.Date(2026, 11, 17, 4, 45, 33, 0, time.Local)
	if err := os.WriteFile(path+"_2026-11-17_04-45-33.bak", []byte("some older backup"), 0644); err != nil {
		t.Fatal(err)
	}
	backup, err := backupAndReplace(r.repairs[0], stamp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(backup, "_002.bak") {
		t.Fatal(backup)
	}
}

func TestNNFixTwoTextureFiles(t *testing.T) {
	ini := `[TextureOverrideFischlHead]
hash = abcdef12
match_first_index = 0
ib = ResourceFischlHeadIB
ps-t0 = ResourceFischlHeadDiffuse
ps-t1 = ResourceFischlHeadLightMap
[TextureOverrideFischlBody]
hash = abcdef12
match_first_index = 500
ib = ResourceFischlBodyIB
ps-t0 = ResourceFischlBodyDiffuse
ps-t1 = ResourceFischlBodyLightMap
[ResourceFischlHeadDiffuse]
filename = FischlHeadDiffuse.dds
[ResourceFischlHeadLightMap]
filename = FischlHeadLightMap.dds
[ResourceFischlBodyDiffuse]
filename = FischlBodyDiffuse.dds
[ResourceFischlBodyLightMap]
filename = FischlBodyLightMap.dds
`
	path := fixture(t, ini, "FischlHeadDiffuse.dds", "FischlHeadLightMap.dds", "FischlBodyDiffuse.dds", "FischlBodyLightMap.dds")
	got, changes, manual := transformINI(path, []byte(ini))
	if len(changes) != 2 || len(manual) != 0 || strings.Count(string(got), nnfixRun) != 2 {
		t.Fatalf("unexpected NNFix: %d %v %v", len(got), changes, manual)
	}
	second, ch, _ := transformINI(path, got)
	if len(second) > 0 || len(ch) > 0 {
		t.Fatalf("NNFix not idempotent")
	}
}

func TestHeadOnlyWeaponSkipped(t *testing.T) {
	ini := `[TextureOverrideSacrificialFragmentsHead]
hash = abcdef12
match_first_index = 0
ib = ResourceSacrificialFragmentsHeadIB
ps-t0 = ResourceSacrificialFragmentsHeadDiffuse
ps-t1 = ResourceSacrificialFragmentsHeadLightMap
[ResourceSacrificialFragmentsHeadDiffuse]
filename = SacrificialFragmentsHeadDiffuse.dds
[ResourceSacrificialFragmentsHeadLightMap]
filename = SacrificialFragmentsHeadLightMap.dds
`
	path := fixture(t, ini, "SacrificialFragmentsHeadDiffuse.dds", "SacrificialFragmentsHeadLightMap.dds")
	patched, changes, manual := transformINI(path, []byte(ini))
	if len(patched) > 0 || len(changes) > 0 || len(manual) == 0 {
		t.Fatalf("weapon changed %v %v", changes, manual)
	}
}

func TestTwoTextureCustomRunSkipped(t *testing.T) {
	ini := `[TextureOverrideFischlBody]
hash = abcdef12
match_first_index = 0
ib = ResourceFischlBodyIB
ps-t0 = ResourceFischlBodyDiffuse
ps-t1 = ResourceFischlBodyLightMap
run = CommandListCustomAlpha
[ResourceFischlBodyDiffuse]
filename = FischlBodyDiffuse.dds
[ResourceFischlBodyLightMap]
filename = FischlBodyLightMap.dds
`
	path := fixture(t, ini, "FischlBodyDiffuse.dds", "FischlBodyLightMap.dds")
	patched, changes, manual := transformINI(path, []byte(ini))
	if len(patched) > 0 || len(changes) > 0 || len(manual) == 0 {
		t.Fatalf("custom run changed %v %v", changes, manual)
	}
}

func TestUnknownTexFxSkipped(t *testing.T) {
	ini := `[TextureOverrideUnknownBody]
hash = abcdef12
match_first_index = 0
ib = ResourceUnknownBodyIB
ps-t0 = ResourceUnknownBodyNormalMap
ps-t1 = ResourceUnknownBodyDiffuse
ps-t2 = ResourceUnknownBodyLightMap
ps-t69 = ResourceUnknownBodyTexfx
run = CommandList\TexFx\Transparency.1
if $skin == 1
    drawindexed = 123, 0, 0
endif
[ResourceUnknownBodyNormalMap]
filename = UnknownBodyNormalMap.dds
[ResourceUnknownBodyDiffuse]
filename = UnknownBodyDiffuse.dds
[ResourceUnknownBodyLightMap]
filename = UnknownBodyLightMap.dds
`
	path := fixture(t, ini, "UnknownBodyNormalMap.dds", "UnknownBodyDiffuse.dds", "UnknownBodyLightMap.dds")
	patched, changes, manual := transformINI(path, []byte(ini))
	if len(patched) > 0 || len(changes) > 0 || len(manual) == 0 {
		t.Fatalf("TexFx wrongly auto-modified %v %v", changes, manual)
	}
}

func TestKnownCitlaliIsExactMatchOnly(t *testing.T) {
	patched, err := recipeFiles.ReadFile("known_fixes/Citlali_high_priest_verified_v2.ini")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "Citlali.ini")
	if !isKnownPatchedCitlali(p, patched) {
		t.Fatal("known patch hash mismatch")
	}
	if x, _, ok := knownRecipe(p, patched); ok || len(x) > 0 {
		t.Fatal("patched file was patched again")
	}
	// Byte difference must prohibit the known recipe. Real unrelated Citlali mods are not interchangeable.
	different := append([]byte(nil), patched...)
	different[len(different)-1] ^= 1
	if _, _, ok := knownRecipe(p, different); ok {
		t.Fatal("non-exact recipe match")
	}
}

func TestSimpleTransparencyNNFix(t *testing.T) {
	ini := `[TextureOverrideEulaHead]
hash = 660399d1
match_first_index = 0
ib = ResourceEulaHeadIB
ps-t0 = ResourceEulaHeadDiffuse
ps-t1 = ResourceEulaHeadLightMap
[TextureOverrideEulaDress]
hash = 660399d1
match_first_index = 11349
ib = ResourceEulaDressIB
ps-t0 = ResourceEulaDressDiffuse
ps-t1 = ResourceEulaDressLightMap
run = CustomShaderTransparency
[ResourceEulaHeadDiffuse]
filename = EulaHeadDiffuse.dds
[ResourceEulaHeadLightMap]
filename = EulaHeadLightMap.dds
[ResourceEulaDressDiffuse]
filename = EulaDressDiffuse.dds
[ResourceEulaDressLightMap]
filename = EulaDressLightMap.dds
[CustomShaderTransparency]
blend = ADD BLEND_FACTOR INV_BLEND_FACTOR
blend_factor[0] = 0.5
blend_factor[1] = 0.5
blend_factor[2] = 0.5
blend_factor[3] = 1
handling = skip
drawindexed = auto
`
	path := fixture(t, ini, "EulaHeadDiffuse.dds", "EulaHeadLightMap.dds", "EulaDressDiffuse.dds", "EulaDressLightMap.dds")
	patched, changes, manual := transformINI(path, []byte(ini))
	if len(changes) != 2 || len(manual) > 0 {
		t.Fatalf("changes=%v manual=%v", changes, manual)
	}
	if strings.Count(string(patched), nnfixRun) != 2 {
		t.Fatalf("missing NNFix: %q", patched)
	}
	if !strings.Contains(string(patched), "ps-t1 = ResourceEulaDressLightMap\n"+nnfixRun+"\nrun = CustomShaderTransparency") {
		t.Fatal("wrong instruction order")
	}
	patched2, changes2, _ := transformINI(path, patched)
	if len(patched2) > 0 || len(changes2) > 0 {
		t.Fatal("not idempotent")
	}
}

func TestUnsafeCustomTransparencyStillSkipped(t *testing.T) {
	ini := `[TextureOverrideEulaHead]
hash = 660399d1
match_first_index = 0
ib = ResourceEulaHeadIB
ps-t0 = ResourceEulaHeadDiffuse
ps-t1 = ResourceEulaHeadLightMap
run = CustomShaderTransparency
[TextureOverrideEulaBody]
hash = 660399d1
[ResourceEulaHeadDiffuse]
filename = EulaHeadDiffuse.dds
[ResourceEulaHeadLightMap]
filename = EulaHeadLightMap.dds
[CustomShaderTransparency]
blend = ADD SRC_ALPHA INV_SRC_ALPHA
handling = skip
drawindexed = auto
`
	path := fixture(t, ini, "EulaHeadDiffuse.dds", "EulaHeadLightMap.dds")
	patched, changes, manual := transformINI(path, []byte(ini))
	if len(patched) > 0 || len(changes) > 0 || len(manual) == 0 {
		t.Fatalf("unsafe shader modified: %v %v", changes, manual)
	}
}
