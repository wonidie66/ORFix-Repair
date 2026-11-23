package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func genericFourSlotFixture(character, hash string, indices [3]int) (string, []string) {
	parts := []string{"Head", "Body", "Dress"}
	var b strings.Builder
	files := []string{}
	b.WriteString("; Generic four-slot character material fixture\n\n")
	b.WriteString("[TextureOverride" + character + "IB]\nhash = " + hash + "\nhandling = skip\ndrawindexed = auto\n\n")
	for i, part := range parts {
		stem := "Resource" + character + part
		b.WriteString("[TextureOverride" + character + part + "]\n")
		b.WriteString("hash = " + hash + "\n")
		b.WriteString("match_first_index = " + fmtInt(indices[i]) + "\n")
		b.WriteString("ib = " + stem + "IB\n")
		b.WriteString("ps-t0 = " + stem + "Diffuse\n")
		b.WriteString("ps-t1 = " + stem + "LightMap\n")
		b.WriteString("ps-t2 = " + stem + "MetalMap\n")
		b.WriteString("ps-t3 = " + stem + "ShadowRamp\n\n")
		b.WriteString("[" + stem + "IB]\nfilename = " + character + part + ".ib\n")
		files = append(files, character+part+".ib")
		for _, suffix := range []string{"Diffuse", "LightMap", "MetalMap", "ShadowRamp"} {
			ext := ".dds"
			if suffix == "ShadowRamp" {
				ext = ".jpg"
			}
			b.WriteString("[" + stem + suffix + "]\nfilename = " + character + part + suffix + ext + "\n")
			files = append(files, character+part+suffix+ext)
		}
		b.WriteString("\n")
	}
	return b.String(), files
}

func fmtInt(v int) string {
	return strconv.Itoa(v)
}

func TestGenericFourSlotNNFixMatchesDifferentCharactersAndMeshSignatures(t *testing.T) {
	cases := []struct {
		name    string
		hash    string
		indices [3]int
	}{
		{name: "Aster", hash: "9a73bc10", indices: [3]int{17, 2841, 7913}},
		{name: "Lynette", hash: "17cd209a", indices: [3]int{120, 5500, 12345}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ini, files := genericFourSlotFixture(tc.name, tc.hash, tc.indices)
			path := createINI(t, ini, files...)
			patched, changes, manual := transformEnhanced(path, []byte(ini), "enhanced")
			if len(changes) != 3 || len(manual) != 0 || bytes.Count(patched, []byte(nnfixRun)) != 3 {
				t.Fatalf("expected three generic four-slot NNFix insertions; changes=%v manual=%v", changes, manual)
			}
			if strings.ReplaceAll(string(patched), "\n"+nnfixRun, "") != ini {
				t.Fatal("generic four-slot rule changed content other than the three NNFix calls")
			}
			second, secondChanges, secondManual := transformEnhanced(path, patched, "enhanced")
			if len(second) != 0 || len(secondChanges) != 0 || len(secondManual) != 0 {
				t.Fatalf("generic four-slot repair is not idempotent: changes=%v manual=%v", secondChanges, secondManual)
			}
		})
	}
}

func TestGenericFourSlotNNFixRejectsMissingResourceFile(t *testing.T) {
	ini, files := genericFourSlotFixture("Aster", "9a73bc10", [3]int{17, 2841, 7913})
	path := createINI(t, ini, files...)
	missing := filepath.Join(filepath.Dir(path), "AsterBodyMetalMap.dds")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	patched, changes, manual := transformEnhanced(path, []byte(ini), "enhanced")
	if len(changes) != 2 || bytes.Count(patched, []byte(nnfixRun)) != 2 || len(manual) == 0 {
		t.Fatalf("only the part with the missing file should be skipped: changes=%v manual=%v", changes, manual)
	}
}

func TestGenericFourSlotNNFixRejectsWrongSemanticSlot(t *testing.T) {
	ini, files := genericFourSlotFixture("Aster", "9a73bc10", [3]int{17, 2841, 7913})
	ini = strings.Replace(ini, "ps-t2 = ResourceAsterBodyMetalMap", "ps-t2 = ResourceAsterBodyNormalMap", 1)
	path := createINI(t, ini, files...)
	patched, changes, manual := transformEnhanced(path, []byte(ini), "enhanced")
	if len(changes) != 2 || bytes.Count(patched, []byte(nnfixRun)) != 2 || len(manual) == 0 {
		t.Fatalf("only the part with the wrong slot semantic should be skipped: changes=%v manual=%v", changes, manual)
	}
}

func TestGenericFourSlotNNFixRejectsMismatchedStem(t *testing.T) {
	ini, files := genericFourSlotFixture("Aster", "9a73bc10", [3]int{17, 2841, 7913})
	ini = strings.Replace(ini, "ps-t3 = ResourceAsterBodyShadowRamp", "ps-t3 = ResourceOtherBodyShadowRamp", 1)
	ini += "[ResourceOtherBodyShadowRamp]\nfilename = OtherShadowRamp.jpg\n"
	files = append(files, "OtherShadowRamp.jpg")
	path := createINI(t, ini, files...)
	patched, changes, manual := transformEnhanced(path, []byte(ini), "enhanced")
	if len(changes) != 2 || bytes.Count(patched, []byte(nnfixRun)) != 2 || len(manual) == 0 {
		t.Fatalf("only the part with the mismatched resource stem should be skipped: changes=%v manual=%v", changes, manual)
	}
}

func TestGenericFourSlotNNFixRejectsFifthSlot(t *testing.T) {
	ini, files := genericFourSlotFixture("Aster", "9a73bc10", [3]int{17, 2841, 7913})
	ini = strings.Replace(ini, "ps-t3 = ResourceAsterBodyShadowRamp\n", "ps-t3 = ResourceAsterBodyShadowRamp\nps-t4 = ResourceAsterBodyExtra\n", 1)
	path := createINI(t, ini, files...)
	patched, changes, manual := transformEnhanced(path, []byte(ini), "enhanced")
	if len(changes) != 2 || bytes.Count(patched, []byte(nnfixRun)) != 2 || len(manual) == 0 {
		t.Fatalf("only the part with the fifth texture slot should be skipped: changes=%v manual=%v", changes, manual)
	}
}

func TestGenericFourSlotNNFixPreservesExistingCalls(t *testing.T) {
	ini, files := genericFourSlotFixture("Aster", "9a73bc10", [3]int{17, 2841, 7913})
	ini = strings.Replace(ini, "ps-t3 = ResourceAsterHeadShadowRamp\n", "ps-t3 = ResourceAsterHeadShadowRamp\n"+nnfixRun+"\n", 1)
	path := createINI(t, ini, files...)
	patched, changes, manual := transformEnhanced(path, []byte(ini), "enhanced")
	if len(changes) != 2 || len(manual) != 0 || bytes.Count(patched, []byte(nnfixRun)) != 3 {
		t.Fatalf("expected only missing Body/Dress calls: changes=%v manual=%v", changes, manual)
	}
}

func TestGenericFourSlotNNFixIntegratedInEnhancedScanAndDiagnostics(t *testing.T) {
	ini, files := genericFourSlotFixture("Aster", "9a73bc10", [3]int{17, 2841, 7913})
	path := createINI(t, ini, files...)
	report := scan(filepath.Dir(path), "enhanced")
	if len(report.repairs) != 1 {
		t.Fatalf("expected one repair plan, got %d; manual=%v errors=%v", len(report.repairs), report.manual, report.errors)
	}
	if bytes.Count(report.repairs[0].patched, []byte(nnfixRun)) != 3 {
		t.Fatal("enhanced scan did not include all generic four-slot NNFix calls")
	}
	findings := strings.Join(diagnoseINI(path, []byte(ini)), "\n")
	if !strings.Contains(findings, "四槽位 Diffuse + LightMap + MetalMap + ShadowRamp") {
		t.Fatalf("diagnostics did not describe the generic four-slot layout: %s", findings)
	}
}
