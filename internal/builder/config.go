package builder

import (
	"encoding/json"
	"os"

	"github.com/alecthomas/chroma/styles"
	"github.com/danprince/sietch/internal/errors"
)

type Config struct {
	SyntaxColor string
	Framework   string
	DateFormat  string
}

var defaultConfig = Config{
	SyntaxColor: "algol_nu",
	Framework:   "vanilla",
	DateFormat:  "2006-1-2",
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

	if _, ok := frameworkMap[c.Framework]; !ok {
		allowed := []string{}

		for s := range frameworkMap {
			allowed = append(allowed, s)
		}

		return errors.ConfigError{
			File:    file,
			Key:     "Framework",
			Value:   c.Framework,
			Allowed: allowed,
		}
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

	return nil
}
