package gou

// Lang for applying a language pack
func (mod *Model) Lang(trans func(widget string, inst string, value *string) bool) {
	trans("model", mod.Name, &mod.MetaData.Name)
	trans("model", mod.Name, &mod.MetaData.Table.Comment)

	for idx := range mod.MetaData.Columns {
		trans("model", mod.Name, &mod.MetaData.Columns[idx].Label)
		trans("model", mod.Name, &mod.MetaData.Columns[idx].Comment)
	}
}
