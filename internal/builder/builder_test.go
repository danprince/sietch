package builder

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/danprince/sietch/internal/errors"
)

func compareLineForLine(t *testing.T, expectName, actualName, expectSrc, actualSrc string) {
	t.Helper()

	actualSrc = strings.TrimSpace(actualSrc)
	expectSrc = strings.TrimSpace(expectSrc)
	expectLines := strings.Split(expectSrc, "\n")
	actualLines := strings.Split(actualSrc, "\n")

	if len(expectLines) != len(actualLines) {
		t.Fatalf(`files had a different number of lines:
%s (%d lines):
%s

%s (%d lines):
%s
`, expectName, len(expectLines), expectSrc, actualName, len(actualLines), actualSrc)
	}

	for index := range expectLines {
		actualLine := actualLines[index]
		expectLine := expectLines[index]
		if actualLine != expectLine {
			t.Fatalf(`files were not equal: %s and %s
line %d did not match:
expect: %s
actual: %s`, expectName, actualName, index+1, expectLine, actualLine)
		}
	}
}

// Files in this list will be compared line by line, other files will be
// treated as binary and have their bytes compared directly instead.
var lineByLineFileExts = map[string]bool{
	".js":   true,
	".css":  true,
	".html": true,
	".txt":  true,
}

func TestFixtures(t *testing.T) {
	cwd, _ := os.Getwd()
	fallbackTemplateFile := path.Join(cwd, "testdata/template.html")
	fixturesDir := path.Join(cwd, "testdata/fixtures")
	dirents, err := os.ReadDir(fixturesDir)

	if err != nil {
		t.Fatal(err)
	}

	for _, dirent := range dirents {
		if !dirent.IsDir() {
			t.Fatalf("fixture was not a dir: %s", dirent.Name())
		}

		name := dirent.Name()

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inputDir := path.Join(fixturesDir, name)
			templateFile := path.Join(inputDir, "_template.html")
			expectErrorFile := path.Join(inputDir, "_expect_error.txt")
			expectDir := path.Join(inputDir, "_expect")
			actualDir := path.Join(inputDir, "_site")

			// Attempt to remove the actualDir before we start testing so that
			// there's nothing stale hanging around between tests.
			os.RemoveAll(actualDir)

			t.Cleanup(func() {
				// If the test failed, leave the dir around for debugging
				if !t.Failed() {
					os.RemoveAll(actualDir)
				}
			})

			builder := New(inputDir, Production)
			builder.OutDir = actualDir
			// Disable minification in tests to keep the output readable
			builder.minify = false

			if _, err := os.Stat(templateFile); err != nil {
				// If there wasn't a template file in the dir, use the default one for
				// fixtures.
				builder.templateFile = fallbackTemplateFile
			}

			if buildErr := builder.Build(); buildErr != nil {
				// If the build fails, check whether there is an expected error file and
				// compare it.
				if expectError, err := os.ReadFile(expectErrorFile); err == nil {
					name, _ := filepath.Rel(inputDir, expectErrorFile)
					errStr := errors.NoColor(buildErr)
					compareLineForLine(t, name, "<actual error>", errStr, string(expectError))
					// If compareLineForLine didn't fail, then it means the error matched and
					// we can end the test here.
					return
				} else {
					// Otherwise, just fail the whole test
					t.Fatal(buildErr)
				}
			}

			// Ensure that there is an _expect to compare against
			if _, err := os.Stat(expectDir); err != nil {
				t.Fatal(err)
			}

			// Check each file in the expect dir lines up with an equivalent file in
			// the actual dir.
			filepath.WalkDir(expectDir, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					t.Error(err)
				}

				// Don't bother comparing the assets dir, there's not much value in comparing
				// generated content.
				if d.IsDir() && d.Name() == "_assets" {
					return filepath.SkipDir
				}

				rel, _ := filepath.Rel(expectDir, p)
				expectPath := p
				actualPath := path.Join(actualDir, rel)
				actualInfo, err := os.Stat(actualPath)

				if err != nil {
					t.Error(err)
				}

				if d.IsDir() && !actualInfo.IsDir() {
					t.Errorf("expected %s to be a dir", actualPath)
				}

				if d.IsDir() && actualInfo.IsDir() {
					return err
				}

				expectContents, err := os.ReadFile(expectPath)

				if err != nil {
					t.Error(err)
				}

				actualContents, err := os.ReadFile(actualPath)

				if err != nil {
					t.Error(err)
				}

				ext := path.Ext(expectPath)
				lineByLine, ok := lineByLineFileExts[ext]

				if ok && lineByLine {
					actualRel, _ := filepath.Rel(inputDir, actualPath)
					expectRel, _ := filepath.Rel(inputDir, expectPath)
					compareLineForLine(t, expectRel, actualRel, string(expectContents), string(actualContents))
				} else {
					if !reflect.DeepEqual(actualContents, expectContents) {
						t.Errorf("files were not equal: %s and %s", expectPath, actualPath)
					}
				}

				return err
			})

			// Check whether there were any files in the output that we weren't
			// expecting to see.
			filepath.WalkDir(actualDir, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					t.Error(err)
				}

				// Ignore generated content
				if d.IsDir() && d.Name() == "_assets" {
					return filepath.SkipDir
				}

				rel, _ := filepath.Rel(actualDir, p)
				expectPath := path.Join(expectDir, rel)
				_, err = os.Stat(expectPath)

				if os.IsNotExist(err) {
					t.Errorf("unexpected file in output: %s", p)
				} else if err != nil {
					t.Error(err)
				}

				return err
			})
		})
	}
}
