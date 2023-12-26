package v8

// func TestProcessScripts(t *testing.T) {
// 	prepare(t)
// 	time.Sleep(20 * time.Millisecond)
// 	assert.Equal(t, 2, isolates.Len)
// 	assert.Equal(t, 2, len(chIsoReady))

// 	p, err := process.Of("scripts.runtime.basic.Hello", "world")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	value, err := p.Exec()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	assert.Equal(t, "world", value)

// 	p, err = process.Of("scripts.runtime.basic.Error", "world")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = p.Exec()
// 	assert.Contains(t, err.Error(), "at callStackTest")
// 	assert.Contains(t, err.Error(), "at Error")

// }

// func TestProcessScriptsRoot(t *testing.T) {
// 	prepare(t)
// 	time.Sleep(20 * time.Millisecond)
// 	assert.Equal(t, 2, isolates.Len)
// 	assert.Equal(t, 2, len(chIsoReady))

// 	p, err := process.Of("studio.runtime.basic.Hello", "world")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	value, err := p.Exec()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	assert.Equal(t, "world", value)
// }
