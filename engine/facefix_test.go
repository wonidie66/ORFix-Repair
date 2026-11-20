package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFaceFixture(t *testing.T, text string) (string, string) {
	t.Helper()
	root := t.TempDir()
	ini := filepath.Join(root, "FaceMod.ini")
	if err := os.WriteFile(filepath.Join(root, "CharacterFaceHeadDiffuse.dds"), []byte{1, 2, 3, 4}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ini, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	return root, ini
}

func TestFaceFixDirectBindingIsOptionalAndTargeted(t *testing.T) {
	text := `[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
ps-t0 = ResourceCharacterFaceHeadDiffuse

[TextureOverrideCharacterHead]
hash = 11111111
ps-t0 = ResourceCharacterHeadDiffuse

[ResourceCharacterFaceHeadDiffuse]
filename = CharacterFaceHeadDiffuse.dds
`
	root, ini := writeFaceFixture(t, text)
	original, _ := os.ReadFile(ini)
	analysis := analyzeFaceINI(ini, original)
	if len(analysis.findings) != 1 || len(analysis.changes) != 1 {
		t.Fatalf("expected one optional face finding/change, got %+v", analysis)
	}
	if !bytes.Contains(analysis.patched, []byte("this = ResourceCharacterFaceHeadDiffuse")) {
		t.Fatal("face binding was not converted to this")
	}
	if !bytes.Contains(analysis.patched, []byte("ps-t0 = ResourceCharacterHeadDiffuse")) {
		t.Fatal("non-face texture binding was modified")
	}
	if bytes.Contains(original, []byte("this = ResourceCharacterFaceHeadDiffuse")) {
		t.Fatal("analysis unexpectedly mutated source bytes")
	}
	plans, findings, manual, errs := scanFaceFix(root)
	if len(plans) != 1 || len(findings) != 1 || len(manual) != 0 || len(errs) != 0 {
		t.Fatalf("unexpected face scan: plans=%d findings=%v manual=%v errs=%v", len(plans), findings, manual, errs)
	}
	// The regular enhanced scan reports risk but does not apply Face Fix.
	r := scan(root, "enhanced")
	if len(r.faceFindings) != 1 {
		t.Fatalf("normal scan should report one face risk: %v", r.faceFindings)
	}
	stillOriginal, _ := os.ReadFile(ini)
	if !bytes.Equal(stillOriginal, original) {
		t.Fatal("normal scan must not modify face configuration")
	}
}

func TestFaceFixFollowsNestedCommandLists(t *testing.T) {
	text := `[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
run = CommandListFaceEntry

[CommandListFaceEntry]
run = CommandListFaceBinding

[CommandListFaceBinding]
ps-t0 = ResourceCharacterFaceHeadDiffuse

[ResourceCharacterFaceHeadDiffuse]
filename = CharacterFaceHeadDiffuse.dds
`
	_, ini := writeFaceFixture(t, text)
	b, _ := os.ReadFile(ini)
	a := analyzeFaceINI(ini, b)
	if len(a.changes) != 1 || !bytes.Contains(a.patched, []byte("this = ResourceCharacterFaceHeadDiffuse")) {
		t.Fatalf("nested face command list was not fixed: %+v", a)
	}
}

func TestFaceFixDoesNotModifySharedCommandList(t *testing.T) {
	text := `[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
run = CommandListSharedDiffuse

[TextureOverrideCharacterBody]
hash = 11111111
run = CommandListSharedDiffuse

[CommandListSharedDiffuse]
ps-t0 = ResourceCharacterFaceHeadDiffuse

[ResourceCharacterFaceHeadDiffuse]
filename = CharacterFaceHeadDiffuse.dds
`
	_, ini := writeFaceFixture(t, text)
	b, _ := os.ReadFile(ini)
	a := analyzeFaceINI(ini, b)
	if len(a.findings) != 1 || len(a.manual) != 1 || len(a.changes) != 0 || len(a.patched) != 0 {
		t.Fatalf("shared command list must be manual-only: %+v", a)
	}
}

func TestFaceFixLeavesExistingThisAlone(t *testing.T) {
	text := `[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
this = ResourceCharacterFaceHeadDiffuse

[ResourceCharacterFaceHeadDiffuse]
filename = CharacterFaceHeadDiffuse.dds
`
	_, ini := writeFaceFixture(t, text)
	b, _ := os.ReadFile(ini)
	a := analyzeFaceINI(ini, b)
	if len(a.findings) != 0 || len(a.changes) != 0 || len(a.patched) != 0 {
		t.Fatalf("already-correct this binding should not be flagged: %+v", a)
	}
}

func TestFaceFixRequiresExistingDiffuseResource(t *testing.T) {
	text := `[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
ps-t0 = ResourceCharacterFaceHeadDiffuse

[ResourceCharacterFaceHeadDiffuse]
filename = missing.dds
`
	root, ini := writeFaceFixture(t, text)
	os.Remove(filepath.Join(root, "CharacterFaceHeadDiffuse.dds"))
	b, _ := os.ReadFile(ini)
	a := analyzeFaceINI(ini, b)
	if len(a.findings) != 1 || len(a.manual) != 1 || len(a.changes) != 0 {
		t.Fatalf("missing texture must be manual only: %+v", a)
	}
}

func TestFaceFixPreservesCRLFAndRollback(t *testing.T) {
	text := strings.ReplaceAll(`[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
  ps-t0 = ResourceCharacterFaceHeadDiffuse ; keep comment

[ResourceCharacterFaceHeadDiffuse]
filename = CharacterFaceHeadDiffuse.dds
`, "\n", "\r\n")
	root, ini := writeFaceFixture(t, text)
	original, _ := os.ReadFile(ini)
	plans, _, _, _ := scanFaceFix(root)
	if len(plans) != 1 {
		t.Fatalf("expected one plan, got %d", len(plans))
	}
	now := time.Now()
	entries, backups, err := applyGroup(plans[0], now)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(backups) != 1 {
		t.Fatalf("expected one changed file and backup: %d %d", len(entries), len(backups))
	}
	patched, _ := os.ReadFile(ini)
	if !bytes.Contains(patched, []byte("  this = ResourceCharacterFaceHeadDiffuse ; keep comment\r\n")) {
		t.Fatalf("CRLF/comment not preserved: %q", patched)
	}
	entries[0].Path = rel(root, entries[0].Path)
	entries[0].Backup = rel(root, entries[0].Backup)
	if _, err := writeRollback(root, now, entries); err != nil {
		t.Fatal(err)
	}
	n, _ := restoreLatest(root, now.Add(time.Second))
	if n != 1 {
		t.Fatalf("expected rollback one file, got %d", n)
	}
	restored, _ := os.ReadFile(ini)
	if !bytes.Equal(restored, original) {
		t.Fatal("rollback did not restore original INI bytes")
	}
}

func TestFaceFixUnresolvedCommandListIsManualOnly(t *testing.T) {
	text := `[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
run = CommandListFaceDiffuseHandler

[ResourceCharacterFaceHeadDiffuse]
filename = CharacterFaceHeadDiffuse.dds
`
	_, ini := writeFaceFixture(t, text)
	b, _ := os.ReadFile(ini)
	a := analyzeFaceINI(ini, b)
	if len(a.findings) != 1 || len(a.manual) != 1 || len(a.changes) != 0 || len(a.patched) != 0 {
		t.Fatalf("unresolved face CommandList must be manual-only: %+v", a)
	}
}

func TestFaceFixIgnoresCommentedPS0(t *testing.T) {
	text := `[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
; ps-t0 = ResourceCharacterFaceHeadDiffuse
this = ResourceCharacterFaceHeadDiffuse

[ResourceCharacterFaceHeadDiffuse]
filename = CharacterFaceHeadDiffuse.dds
`
	_, ini := writeFaceFixture(t, text)
	b, _ := os.ReadFile(ini)
	a := analyzeFaceINI(ini, b)
	if len(a.findings) != 0 || len(a.changes) != 0 || len(a.patched) != 0 {
		t.Fatalf("commented ps-t0 plus this must not be flagged: %+v", a)
	}
}
