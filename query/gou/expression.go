package gou

// UnmarshalJSON for json marshalJSON
func (exp *Expression) UnmarshalJSON(data []byte) error {
	exp.string = string(data)
	return nil
}

// MarshalJSON for json marshalJSON
func (exp *Expression) MarshalJSON() ([]byte, error) {
	return []byte(exp.string), nil
}

// NewExpression 创建一个表达式
func NewExpression(s string) *Expression {
	return &Expression{
		string: s,
	}
}

// ToString for json marshalJSON
func (exp Expression) ToString() string {
	return exp.string
}
