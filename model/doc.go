package model

import (
	_ "embed"
	"github.com/yaoapp/gou/doc"
)

//go:embed doc.yml
var docYAML []byte

//go:embed doc_model.yml
var docModelYAML []byte

func init() {
	doc.LoadYAML(docYAML)
	doc.LoadYAML(docModelYAML)
}
