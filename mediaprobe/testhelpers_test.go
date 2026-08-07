package mediaprobe

import (
	"errors"
	"io"
)

var errSeekFail = errors.New("seek failed")

type errSeeker struct {
	r      io.ReadSeeker
	failAt []int64
}

func (e *errSeeker) Read(p []byte) (int, error) {
	return e.r.Read(p)
}

func (e *errSeeker) Seek(offset int64, whence int) (int64, error) {
	pos, err := e.r.Seek(offset, whence)
	if err != nil {
		return pos, err
	}
	for _, at := range e.failAt {
		if pos == at {
			return pos, errSeekFail
		}
	}
	return pos, nil
}

func newErrSeeker(r io.ReadSeeker, failAt ...int64) *errSeeker {
	return &errSeeker{r: r, failAt: failAt}
}
