package types

// GetText returns the text of the segment
func (segment *Segment) GetText() string {
	return segment.Text
}

// ================================================================================================
// SegmentTree Methods
// ================================================================================================

// IsRoot returns true if this node has no parent
func (st *SegmentTree) IsRoot() bool {
	return st.Parent == nil
}

// GetParentChain returns the parent chain from current segment to root
func (st *SegmentTree) GetParentChain() []*Segment {
	var parents []*Segment
	current := st.Parent

	// Walk up the parent chain
	for current != nil {
		if current.Segment != nil {
			parents = append(parents, current.Segment)
			current = current.Parent
		} else {
			break
		}
	}

	return parents
}
