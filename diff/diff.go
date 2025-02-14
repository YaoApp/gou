package diff

import (
	"fmt"
	"strings"

	diffmatchpatch "github.com/sergi/go-diff/diffmatchpatch"
)

// ReplacePatch struct
type ReplacePatch struct {
	Search  string
	Replace string
}

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

// Replace Replace a text with a patch
// Format:
// <<<<<<< SEARCH
// some text
// =======
// some other text
// >>>>>>> REPLACE
// Text Text
// <<<<<<< SEARCH
// some text 2
// =======
// some other text 2
// >>>>>>> REPLACE
// Text Text 2
func Replace(text string, patch string) (string, error) {
	patches := parsePatch(patch)
	notFound := []string{}
	result := text

	for _, p := range patches {
		if !strings.Contains(result, p.Search) {
			notFound = append(notFound, p.Search)
		}
	}

	if len(notFound) > 0 {
		return text, fmt.Errorf("search text not found: %s", strings.Join(notFound, ", "))
	}

	for _, p := range patches {
		result = strings.Replace(result, p.Search, p.Replace, 1)
	}
	return result, nil
}

// parsePatch Parse a patch string into a search and replace text
func parsePatch(patch string) []ReplacePatch {
	patches := []ReplacePatch{}
	if patch == "" {
		return patches
	}

	lines := strings.Split(patch, "\n")
	startSearch := false
	startReplace := false
	current := -1

	for _, line := range lines {
		lineTrim := strings.TrimSpace(line)

		// Find the search block
		if lineTrim == "<<<<<<< SEARCH" {
			startSearch = true
			startReplace = false
			patches = append(patches, ReplacePatch{})
			current++
			continue
		}

		// Find the replace block
		if startSearch && lineTrim == "=======" {
			// Remove the trailing newline from search text
			if len(patches[current].Search) > 0 {
				patches[current].Search = patches[current].Search[:len(patches[current].Search)-1]
			}
			startReplace = true
			startSearch = false
			continue
		}

		// End of patch
		if lineTrim == ">>>>>>> REPLACE" {
			// Remove the trailing newline from replace text
			if len(patches[current].Replace) > 0 {
				patches[current].Replace = patches[current].Replace[:len(patches[current].Replace)-1]
			}
			startSearch = false
			startReplace = false
			continue
		}

		// Add line to appropriate section
		if startSearch {
			patches[current].Search += line + "\n"
		} else if startReplace {
			patches[current].Replace += line + "\n"
		}
	}

	return patches
}
