package errors

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-sourcemap/sourcemap"
	"rogchap.com/v8go"
)

var (
	yamlLineErrorRegex      = regexp.MustCompile(`yaml: line (\d+): `)
	templateParseErrorRegex = regexp.MustCompile(`template: .+:(\d+): `)
	templateExecErrorRegex  = regexp.MustCompile(`template: .+:(\d+):(\d+): .+ <(.+?)>: `)
)

const (
	errorColor      = "\033[1;31m"
	errorFocusColor = "\033[1m"
	lineNumberColor = "\033[2m"
	resetColor      = "\033[0m"
	errorHighlight  = "\033[41;1m"
)

// Wrap formats errors to prepend a domain with the error color.
func Wrap(domain string, err error) error {
	return fmt.Errorf("%s%s%s: %s", errorColor, domain, resetColor, err)
}

func relativeToCwd(file string) string {
	if file[0] == '.' || file[0] == '/' {
		cwd, _ := os.Getwd()
		file, _ = filepath.Rel(cwd, file)
	}
	return file
}

// SourceError is an error that directly relates to a file with a line/column
// that we can show to the user for context.
type SourceError struct {
	file       string
	virtual    bool
	line       int
	column     int
	message    string
	details    string
	lineOffset int
	contents   string
}

func (e *SourceError) Error() string {
	file := e.file
	contents := e.contents

	// If no contents were provided, then try to read the file from disk instead.
	if !e.virtual && len(e.contents) == 0 {
		if data, err := os.ReadFile(file); err == nil {
			contents = string(data)
		}
	}

	if path.IsAbs(file) {
		// Turn absolute paths into relative ones that are easier to read.
		file = relativeToCwd(file)
	} else if e.virtual {
		// Use angle brackets to signify virtual files
		file = "<" + file + ">"
	}

	lines := strings.Split(contents, "\n")
	line := e.line - e.lineOffset - 1 // use 0 based lines
	start := line - 3
	end := line + 3

	if start < 0 {
		start = 0
	}

	if end >= len(lines) {
		end = len(lines) - 1
	}

	var sb strings.Builder

	sb.WriteString(errorColor + e.message + resetColor + "\n\n")

	sb.WriteString(errorFocusColor)
	if e.column > 0 {
		sb.WriteString(fmt.Sprintf("%s:%d:%d", file, e.line, e.column))
	} else {
		sb.WriteString(fmt.Sprintf("%s:%d", file, e.line))
	}
	sb.WriteString(resetColor)
	sb.WriteByte('\n')

	for i := start; i <= end; i++ {
		color := lineNumberColor

		if i == line {
			color = errorColor
		}

		n := i + e.lineOffset + 1
		text := lines[i]
		sb.WriteString(fmt.Sprintf("%s%3d%s %s\n", color, n, resetColor, text))

		if i == line && e.column > 0 {
			padding := strings.Repeat(" ", e.column+4)
			sb.WriteString(fmt.Sprintf("%s%s^%s\n", errorColor, padding, resetColor))
		}
	}

	if len(e.details) > 0 {
		sb.WriteString(fmt.Sprintf("\n%s\n", e.details))
	}

	return sb.String()
}

// Determines the 1-based line/column from a string and an offset.
func loc(source string, offset int) (line int, col int) {
	block := source[:offset]
	line = strings.Count(block, "\n") + 1
	col = offset - strings.LastIndexByte(block, '\n')
	return
}

func EsbuildError(result api.BuildResult) error {
	err := result.Errors[0]

	var details strings.Builder

	for _, note := range err.Notes {
		details.WriteString(note.Text)
		details.WriteByte('\n')
	}

	if err.Location == nil {
		return Wrap("esbuild", fmt.Errorf(err.Text))
	}

	virtual := false
	lineOffset := err.Location.Line - 1
	contents := err.Location.LineText

	if source, _ := os.ReadFile(err.Location.File); source != nil {
		contents = string(source)
		lineOffset = 0
	} else {
		virtual = true
	}

	return &SourceError{
		file:       err.Location.File,
		line:       err.Location.Line,
		column:     err.Location.Column,
		contents:   contents,
		lineOffset: lineOffset,
		message:    fmt.Sprintf("esbuild: %s", err.Text),
		details:    details.String(),
		virtual:    virtual,
	}
}

func YamlParseError(err error, file string, contents string) error {
	matches := yamlLineErrorRegex.FindStringSubmatch(err.Error())

	if len(matches) < 2 {
		return err
	}

	line, _ := strconv.Atoi(matches[1])
	msg := yamlLineErrorRegex.ReplaceAllString(err.Error(), "yaml: ")

	return &SourceError{
		file:     file,
		line:     line,
		contents: contents,
		message:  msg,
		details:  `Couldn't parse the front matter from this file.`,
	}
}

func TemplateParseError(err error, file string, contents string, lineOffset int) error {
	matches := templateParseErrorRegex.FindStringSubmatch(err.Error())

	if len(matches) < 2 {
		return err
	}

	line, _ := strconv.Atoi(matches[1])
	msg := templateParseErrorRegex.ReplaceAllString(err.Error(), "")

	return &SourceError{
		message:    fmt.Sprintf("template: %s", msg),
		file:       file,
		line:       line + lineOffset,
		lineOffset: lineOffset,
		contents:   contents,
	}
}

func TemplateExecError(err error, file string, contents string, lineOffset int) error {
	matches := templateExecErrorRegex.FindStringSubmatch(err.Error())

	if len(matches) < 3 {
		return err
	}

	line, _ := strconv.Atoi(matches[1])
	column, _ := strconv.Atoi(matches[2])
	msg := templateExecErrorRegex.ReplaceAllString(err.Error(), "")

	return &SourceError{
		message:    fmt.Sprintf("template: %s", msg),
		file:       file,
		line:       line + lineOffset,
		column:     column,
		lineOffset: lineOffset,
		contents:   contents,
	}
}

func JsonParseError(err error, file string, contents string) error {
	if err, ok := err.(*json.SyntaxError); ok {
		line, column := loc(contents, int(err.Offset))

		return &SourceError{
			message:  fmt.Sprintf("json: %s", err.Error()),
			file:     file,
			line:     line,
			column:   column,
			contents: contents,
		}
	}

	if err, ok := err.(*json.UnmarshalTypeError); ok {
		line, column := loc(contents, int(err.Offset))

		return &SourceError{
			file:     file,
			line:     line,
			column:   column,
			contents: contents,
			message:  fmt.Sprintf("json: %s.%s expected %s", err.Struct, err.Field, err.Type),
		}
	}

	return err
}

var (
	v8LocationRegex   = regexp.MustCompile(`(.+):(\d+):(\d+)`)
	v8StackFrameRegex = regexp.MustCompile(`at\s*(\S*)\s*\(?(.*?):(\d+):(\d+)\)`)
)

func V8Error(err error, name string, source []byte, sourceMap []byte, dir string) error {
	jserr, ok := err.(*v8go.JSError)

	if !ok {
		return err
	}

	// V8 doesn't do sourcemaps, so we need to manually reconstruct the error
	sm, _ := sourcemap.Parse(name, sourceMap)
	location := jserr.Location
	contents := string(source)
	virtual := true

	// The error's location is in the form filename:line:col
	m := v8LocationRegex.FindStringSubmatch(location)
	file := m[1]
	line, _ := strconv.Atoi(m[2])
	column, _ := strconv.Atoi(m[3])

	// Attempt to resolve the line/col using the sourcemap, to find the origin
	if _file, _, _line, _column, ok := sm.Source(line-1, column-1); ok {
		file = path.Join(dir, _file)
		line = _line
		column = _column
		contents = "" // read the file later instead
		virtual = false
	}

	// Fix the individual calls in the stack trace using the sourcemap
	stackTrace := v8StackFrameRegex.ReplaceAllStringFunc(jserr.StackTrace, func(s string) string {
		m := v8StackFrameRegex.FindStringSubmatch(s)
		fn := m[1]
		file := path.Join(dir, m[2])
		line, _ := strconv.Atoi(m[3])
		column, _ := strconv.Atoi(m[4])

		if _file, _, _line, _column, ok := sm.Source(line-1, column-1); ok {
			file = relativeToCwd(path.Join(dir, _file))
			line = _line
			column = _column
		}

		if len(fn) > 0 {
			return fmt.Sprintf("at %s (%s:%d:%d)", fn, file, line, column)
		} else {
			return fmt.Sprintf("at %s:%d:%d", file, line, column)
		}
	})

	return &SourceError{
		file:     file,
		message:  fmt.Sprintf("v8: %s", jserr.Message),
		details:  stackTrace,
		contents: contents,
		line:     line,
		column:   column,
		virtual:  virtual,
	}
}

type ConfigError struct {
	File    string
	Key     string
	Value   string
	Allowed []string
	Message string
}

func (e ConfigError) Error() string {
	cwd, _ := os.Getwd()
	file, _ := filepath.Rel(cwd, e.File)
	sort.Strings(e.Allowed)
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%sconfig error: %s%s\n", errorColor, file, resetColor))
	sb.WriteString(fmt.Sprintf("Invalid value for %s: %s%v%s\n", e.Key, errorColor, e.Value, resetColor))
	sb.WriteString(fmt.Sprintf("Expected one of: %s%s%s", errorFocusColor, strings.Join(e.Allowed, ", "), resetColor))
	return sb.String()
}

var htmlColorCodes = map[string]string{
	errorColor:      `<span style="color: #e41010; font-weight: bold">`,
	lineNumberColor: `<span style="color: #adadad">`,
	errorFocusColor: `<span style="font-weight: bold">`,
	resetColor:      `</span>`,
}

var styles = map[string]string{
	"overflow-x":  "auto",
	"font-family": "Consolas,Menlo,Monaco,monospace",
	"margin":      "32px",
}

func Html(e error) string {
	str := e.Error()
	str = html.EscapeString(str)

	for color, tag := range htmlColorCodes {
		str = strings.ReplaceAll(str, color, tag)
	}

	var style strings.Builder

	for k, v := range styles {
		style.WriteString(fmt.Sprintf("%s:%s;", k, v))
	}

	return fmt.Sprintf(`<pre style="%s">%s</pre>`, style.String(), str)
}
