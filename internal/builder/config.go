package builder

import (
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/alecthomas/chroma/styles"
	"github.com/danprince/sietch/internal/errors"
)

type Config struct {
	SyntaxColor string
	DateFormat  string
	PagesDir    string
}

var defaultConfig = Config{
	SyntaxColor: "algol_nu",
	DateFormat:  "2006-1-2",
	PagesDir:    ".",
}

func (c *Config) load(file string) error {
	data, err := os.ReadFile(file)

	if os.IsNotExist(err) {
		return nil
	}

	err = json.Unmarshal(data, c)

	if err != nil {
		return errors.JsonParseError(err, file, string(data))
	}

	// The "css" theme isn't part of chroma, but we use it to enable the
	// "WithClasses" option internally.
	if _, ok := styles.Registry[c.SyntaxColor]; !ok && c.SyntaxColor != "css" {
		allowed := []string{"css"}

		for s := range styles.Registry {
			allowed = append(allowed, s)
		}

		return errors.ConfigError{
			File:    file,
			Key:     "SyntaxColor",
			Value:   c.SyntaxColor,
			Allowed: allowed,
		}
	}

	if strings.HasPrefix(c.PagesDir, "..") || path.IsAbs(c.PagesDir) || strings.HasPrefix(c.PagesDir, "~") {
		return errors.ConfigError{
			File:    file,
			Key:     "PagesDir",
			Value:   c.PagesDir,
			Message: "The pages directory must be inside the site.",
		}
	}

	return nil
}
