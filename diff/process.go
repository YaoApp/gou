package diff

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessPatch Compare two strings and return the patch string
func ProcessPatch(p *process.Process) interface{} {
	p.ValidateArgNums(3)
	text1 := p.ArgsString(0)
	text2 := p.ArgsString(1)
	checklines := p.ArgsBool(2)
	return PatchString(text1, text2, checklines)
}

// ProcessPatchApply Apply a patch string to text
func ProcessPatchApply(p *process.Process) interface{} {
	p.ValidateArgNums(2)
	text := p.ArgsString(0)
	patch := p.ArgsString(1)
	applied, _, err := PatchApplyString(text, patch)
	if err != nil {
		exception.New("Patch apply error: %s", 500, err).Throw()
	}
	return applied
}

// ProcessReplace Replace text using a patch string
func ProcessReplace(p *process.Process) interface{} {
	p.ValidateArgNums(2)
	text := p.ArgsString(0)
	patch := p.ArgsString(1)
	applied, err := Replace(text, patch)
	if err != nil {
		exception.New("Replace error: %s", 500, err).Throw()
	}
	return applied
}

func init() {
	process.Register("diff.Patch", ProcessPatch)
	process.Register("diff.Apply", ProcessPatchApply)
	process.Register("diff.Replace", ProcessReplace)
}
