package builder

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path"
)

func shortHash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum32())[:4]
}

// Implements a less comparator for sorting for any pair of values. These
// values almost certainly come from the front matter section of pages, so
// we never know their actual type upfront.
func lessAny(a any, b any) bool {
	s1, okS1 := a.(string)
	s2, okS2 := b.(string)
	if okS1 && okS2 {
		return s1 < s2
	}

	i1, okI1 := a.(int)
	i2, okI2 := b.(int)
	if okI1 && okI2 {
		return i1 < i2
	}

	// Can support more cases if they come up.

	return false
}

func copyFile(src string, dst string) error {
	dir := path.Dir(dst)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)

	if err != nil {
		return err
	}

	defer srcFile.Close()

	dstFile, err := os.Create(dst)

	if err != nil {
		return err
	}

	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)

	if err != nil {
		return err
	}

	return nil
}
