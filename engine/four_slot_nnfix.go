package engine

// classifyFourSlotNNFix recognizes one verified semantic material layout,
// independent of character name, IB hash, or match_first_index:
//
//	ps-t0 = *Diffuse
//	ps-t1 = *LightMap
//	ps-t2 = *MetalMap
//	ps-t3 = *ShadowRamp
//
// The four resource references must share the same stem and optional numeric
// variant, and every referenced resource must resolve to a non-empty file.
// Any additional, duplicate, missing, or differently typed slot is rejected.
// This is deliberately not a blanket rule for arbitrary four-slot materials.
func classifyFourSlotNNFix(path string, group []binding, resources map[string]string) (bool, string) {
	hasSlot3OrHigher := false
	for _, b := range group {
		if b.slot >= 3 {
			hasSlot3OrHigher = true
			break
		}
	}
	if !hasSlot3OrHigher {
		return false, ""
	}

	if len(group) != 4 {
		return false, "unsupported additional or duplicate texture slots"
	}

	slots := make(map[int]binding, 4)
	for _, b := range group {
		if b.slot < 0 || b.slot > 3 {
			return false, "unsupported additional texture slot"
		}
		if _, exists := slots[b.slot]; exists {
			return false, "duplicate texture slot assignment"
		}
		slots[b.slot] = b
	}

	expected := []string{"diffuse", "lightmap", "metalmap", "shadowramp"}
	var commonStem, commonVariant string
	for slot, expectedKind := range expected {
		b, exists := slots[slot]
		if !exists {
			return false, "four-slot material layout is incomplete"
		}
		stem, kind, variant, ok := refType(b.ref)
		if !ok || kind != expectedKind {
			return false, "four-slot material types do not match Diffuse/LightMap/MetalMap/ShadowRamp"
		}
		if slot == 0 {
			commonStem, commonVariant = stem, variant
		} else if stem != commonStem || variant != commonVariant {
			return false, "four-slot resource stem or variant mismatch"
		}
		if !fileForRef(path, b.ref, resources) {
			return false, "missing four-slot material resource files"
		}
	}
	return true, ""
}

// isFourSlotNNFixGroup is the boolean form used by diagnostics after the
// classifier has already accepted the group.
func isFourSlotNNFixGroup(path string, group []binding, resources map[string]string) bool {
	matched, _ := classifyFourSlotNNFix(path, group, resources)
	return matched
}
