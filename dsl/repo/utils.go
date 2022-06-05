package repo

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type downloadProcess struct {
	total uint64
	call  func(total uint64)
}

var tempNameRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randName(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = tempNameRunes[rand.Intn(len(tempNameRunes))]
	}
	prefix := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s-%s", prefix, string(b))
}

func tempFile(n int, ext string) (string, error) {
	return makeTempFile(n, ext, 0)
}

func makeTempFile(n int, ext string, times int) (string, error) {
	times++
	if times > 10 {
		return "", fmt.Errorf("reach to max times try to make a temp file")
	}

	tmpfile := filepath.Join(os.TempDir(), fmt.Sprintf("%s.%s", randName(n), ext))
	_, err := os.Stat(tmpfile)
	if errors.Is(err, os.ErrNotExist) {
		return tmpfile, nil
	}
	return makeTempFile(n, ext, times)
}

func (p *downloadProcess) Write(bytes []byte) (int, error) {
	n := len(bytes)
	p.total += uint64(n)
	if p.call != nil {
		p.call(p.total)
	}
	return n, nil
}
