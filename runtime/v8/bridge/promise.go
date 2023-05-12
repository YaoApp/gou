package bridge

// Result returns the result of the promise.
func (p PromiseT) Result() (interface{}, error) {
	promise, err := p.value.AsPromise()
	if err != nil {
		return nil, err
	}

	value := promise.Result()
	defer func() {
		if value != nil {
			value.Release()
		}
	}()

	goValue, err := GoValue(value, p.ctx)
	if err != nil {
		return nil, err
	}

	return goValue, nil
}
