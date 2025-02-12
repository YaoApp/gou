package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestPatch(t *testing.T) {
	tests := []struct {
		name      string
		text1     string
		text2     string
		checkline bool
		wantPatch string
	}{
		{
			name:      "simple addition",
			text1:     "Hello World",
			text2:     "Hello Beautiful World",
			checkline: false,
			wantPatch: "@@ -1,11 +1,21 @@\n Hello \n+Beautiful \n World\n",
		},
		{
			name:      "simple deletion",
			text1:     "The quick brown fox",
			text2:     "The fox",
			checkline: false,
			wantPatch: "@@ -1,19 +1,7 @@\n The \n-quick brown \n fox\n",
		},
		{
			name:      "multiple changes",
			text1:     "Hello\nWorld\nTest",
			text2:     "Hi\nWorld\nTesting",
			checkline: true,
			wantPatch: "@@ -1,9 +1,6 @@\n H\n-ello\n+i\n %0AWor\n@@ -6,8 +6,11 @@\n rld%0ATest\n+ing\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Patch(tt.text1, tt.text2, tt.checkline)
			got := PatchString(tt.text1, tt.text2, tt.checkline)
			assert.Equal(t, tt.wantPatch, got)

			// Test patch application
			applied, results := PatchApply(tt.text1, patches)
			assert.Equal(t, tt.text2, applied)
			for _, result := range results {
				assert.True(t, result)
			}
		})
	}
}

func TestPatchApplyString(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		patch     string
		want      string
		wantError bool
	}{
		{
			name:      "valid patch",
			text:      "Hello World",
			patch:     "@@ -1,11 +1,20 @@\n Hello \n+Beautiful \n World\n",
			want:      "Hello Beautiful World",
			wantError: false,
		},
		{
			name:      "invalid patch format",
			text:      "Hello World",
			patch:     "invalid patch format",
			want:      "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, results, err := PatchApplyString(tt.text, tt.patch)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
				for _, result := range results {
					assert.True(t, result)
				}
			}
		})
	}
}

func TestPatchFrom(t *testing.T) {
	tests := []struct {
		name      string
		patch     string
		wantError bool
	}{
		{
			name:      "valid patch",
			patch:     "@@ -1,11 +1,20 @@\n Hello \n+Beautiful \n World\n",
			wantError: false,
		},
		{
			name:      "invalid patch",
			patch:     "invalid patch format",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches, err := PatchFrom(tt.patch)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, patches)
			}
		})
	}
}

func TestProcessPatch(t *testing.T) {
	// Test case 1: Basic text difference
	text1 := "Hello World"
	text2 := "Hello Beautiful World"
	res, err := process.New("diff.Patch", text1, text2, true).Exec()
	if err != nil {
		t.Fatal(err)
	}
	patch := res.(string)
	assert.Contains(t, patch, "Beautiful")

	// Test case 2: No difference
	text3 := "Hello World"
	text4 := "Hello World"
	res2, err := process.New("diff.Patch", text3, text4, true).Exec()
	if err != nil {
		t.Fatal(err)
	}
	patch2 := res2.(string)
	assert.Equal(t, "", patch2)
}

func TestProcessPatchApply(t *testing.T) {
	// Test case 1: Apply patch to text
	original := "Hello World"
	modified := "Hello Beautiful World"
	patch, err := process.New("diff.Patch", original, modified, true).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Apply the patch
	result, err := process.New("diff.PatchApply", original, patch).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, modified, result)

	// Test case 2: Apply empty patch
	emptyPatch, err := process.New("diff.Patch", original, original, true).Exec()
	if err != nil {
		t.Fatal(err)
	}
	result2, err := process.New("diff.PatchApply", original, emptyPatch).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, original, result2)
}
