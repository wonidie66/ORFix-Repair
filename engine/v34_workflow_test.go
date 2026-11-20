package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestV34FaceFixRequiresUserContinue(t *testing.T) {
	root, ini, origINI, origBlend := makeLegacyNingguangFixture(t)
	filename := filepath.Join(root, "NingguangFaceHeadDiffuse.dds")
	if err := os.WriteFile(filename, []byte{1, 2, 3, 4}, 0644); err != nil {
		t.Fatal(err)
	}
	face := `[TextureOverrideNingguangFaceHeadDiffuse]
hash = 20174ee8
ps-t0 = ResourceNingguangFaceHeadDiffuse
[ResourceNingguangFaceHeadDiffuse]
filename = NingguangFaceHeadDiffuse.dds
`
	original := append(append([]byte{}, origINI...), []byte(face)...)
	if err := os.WriteFile(ini, original, 0644); err != nil {
		t.Fatal(err)
	}
	n := runNormalStage(root, time.Now())
	if n.Modified != 1 || len(n.FaceRisks) != 1 || len(n.RollbackItems) != 2 {
		t.Fatalf("normal outcome %+v", n)
	}
	afterNormal, _ := os.ReadFile(ini)
	if bytes.Contains(afterNormal, []byte("this = ResourceNingguangFaceHeadDiffuse")) {
		t.Fatal("normal stage changed face without user choice")
	}
	if !bytes.Contains(afterNormal, []byte("ps-t0 = ResourceNingguangFaceHeadDiffuse")) {
		t.Fatal("normal stage removed face ps-t0")
	}
	if bytes.Count(afterNormal, []byte(nnfixRun)) != 3 {
		t.Fatal("regular NNFix failed")
	}
	faceResult := runFaceStage(root, n.RollbackItems, time.Now().Add(2*time.Second))
	if faceResult.Modified != 1 || len(faceResult.RollbackItems) != 2 || len(faceResult.Errors) > 0 {
		t.Fatalf("face result %+v", faceResult)
	}
	afterFace, _ := os.ReadFile(ini)
	if !bytes.Contains(afterFace, []byte("this = ResourceNingguangFaceHeadDiffuse")) {
		t.Fatal("face did not change on continue")
	}
	if bytes.Contains(afterFace, []byte("ps-t0 = ResourceNingguangFaceHeadDiffuse")) {
		t.Fatal("old face binding remains")
	}
	nRestored, logs := restoreLatest(root, time.Now().Add(5*time.Second))
	if nRestored != 2 {
		t.Fatalf("merged rollback restored %d, logs=%v", nRestored, logs)
	}
	restoredINI, _ := os.ReadFile(ini)
	restoredBUF, _ := os.ReadFile(filepath.Join(root, "NingguangBlend.buf"))
	if !bytes.Equal(restoredINI, original) || !bytes.Equal(restoredBUF, origBlend) {
		t.Fatal("two-stage rollback did not restore original INI & BUF")
	}
}

func TestV34NormalStageCanFinishWithoutFaceModification(t *testing.T) {
	root, ini := writeFaceFixture(t, `[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
ps-t0 = ResourceCharacterFaceHeadDiffuse
[ResourceCharacterFaceHeadDiffuse]
filename = CharacterFaceHeadDiffuse.dds
`)
	orig, _ := os.ReadFile(ini)
	n := runNormalStage(root, time.Now())
	if len(n.FaceRisks) != 1 {
		t.Fatalf("face warning missing: %+v", n)
	}
	after, _ := os.ReadFile(ini)
	if !bytes.Equal(after, orig) {
		t.Fatal("normal workflow must never modify facial pixels or binding")
	}
}

func TestV34NestedSharedCommandListMustNotAutoPatch(t *testing.T) {
	text := `[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
run = CommandListFaceEntry
[CommandListFaceEntry]
run = CommandListSharedDiffuse
[TextureOverrideCharacterBody]
hash = aaaaaaaa
run = CommandListBodyEntry
[CommandListBodyEntry]
run = CommandListSharedDiffuse
[CommandListSharedDiffuse]
ps-t0 = ResourceCharacterFaceHeadDiffuse
[ResourceCharacterFaceHeadDiffuse]
filename = CharacterFaceHeadDiffuse.dds
`
	_, ini := writeFaceFixture(t, text)
	b, _ := os.ReadFile(ini)
	a := analyzeFaceINI(ini, b)
	if len(a.manual) == 0 || len(a.changes) != 0 || len(a.patched) > 0 {
		t.Fatalf("indirectly shared command list must not auto patch: %+v", a)
	}
	if !strings.Contains(strings.Join(a.manual, " "), "同时被非面部区块调用") {
		t.Fatalf("missing explicit reason: %v", a.manual)
	}
}

func TestV34FaceNonFaceResourceMustNotAutoPatch(t *testing.T) {
	root, ini := writeFaceFixture(t, `[TextureOverrideCharacterFaceHeadDiffuse]
hash = 20174ee8
ps-t0 = ResourceCharacterBodyDiffuse
[ResourceCharacterBodyDiffuse]
filename = CharacterFaceHeadDiffuse.dds
`)
	_ = root
	b, _ := os.ReadFile(ini)
	a := analyzeFaceINI(ini, b)
	if len(a.changes) != 0 || len(a.patched) > 0 {
		t.Fatalf("nonface resource was changed: %+v", a)
	}
}
