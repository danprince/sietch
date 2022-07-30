package main

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/styles"
)

type config struct {
	SyntaxColor string
}

// Validate config settings
func (c *config) validate() error {
	if c.SyntaxColor != "" && c.SyntaxColor != "css" {
		style := styles.Registry[c.SyntaxColor]

		if style == nil {
			var sb strings.Builder

			for name := range styles.Registry {
				if name[0] == c.SyntaxColor[0] {
					sb.WriteString(fmt.Sprintf("- %s\n", name))
				}
			}

			suggestion := ""

			if sb.Len() > 0 {
				suggestion = fmt.Sprintf("\n\nMaybe you meant:\n%s", sb.String())
			}

			return fmt.Errorf(`invalid syntax color: %s%s`, c.SyntaxColor, suggestion)
		}
	}

	return nil
}
