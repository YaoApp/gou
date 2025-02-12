package diff

import (
	diffmatchpatch "github.com/sergi/go-diff/diffmatchpatch"
)

// Patch Compare two strings and return the array of differences
func Patch(text1, text2 string, checklines bool) []diffmatchpatch.Patch {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(text1, text2, checklines)
	patches := dmp.PatchMake(diffs)
	return patches
}

// PatchString Compare two strings and return the array of differences
func PatchString(text1, text2 string, checklines bool) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(text1, text2, checklines)
	patches := dmp.PatchMake(diffs)
	return dmp.PatchToText(patches)
}

// PatchFrom Create a new patch from a text
func PatchFrom(text string) ([]diffmatchpatch.Patch, error) {
	dmp := diffmatchpatch.New()
	return dmp.PatchFromText(text)
}

// PatchApply Apply a patch to a text
func PatchApply(text string, patch []diffmatchpatch.Patch) (string, []bool) {
	dmp := diffmatchpatch.New()
	return dmp.PatchApply(patch, text)
}

// PatchApplyString Apply a text formatted patch to a text
func PatchApplyString(text string, patch string) (string, []bool, error) {
	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(patch)
	if err != nil {
		return "", nil, err
	}
	applied, results := dmp.PatchApply(patches, text)
	return applied, results, nil
}
