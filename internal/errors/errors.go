package errors

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

const (
	errorColor      = "\033[1;31m"
	errorFocusColor = "\033[1m"
	errorLineColor  = "\033[2m"
	resetColor      = "\033[0m"
)

var yamlLineErrorRegex = regexp.MustCompile(`yaml: line (\d+): `)
var templateParseErrorRegex = regexp.MustCompile(`template: .+:(\d+): `)
var templateExecErrorRegex = regexp.MustCompile(`template: .+:(\d+):(\d+): .+ <(.+?)>: `)

type codeFrameError struct {
	summary string
	err     error
	msg     string
	src     string
	line    int
	column  int
	file    string
	offset  int
}

func (e *codeFrameError) Error() string {
	lines := strings.Split(e.src, "\n")
	line := e.line - 1 // make line zero-based
	startLine := line - 3
	endLine := line + 3

	if startLine < 0 {
		startLine = 0
	}

	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	var b strings.Builder

	if e.file != "" {
		file := strings.Replace(e.file, "/", "./", 1)
		lineNumber := e.offset + line + 1
		b.WriteString(fmt.Sprintf("%s%s:%d%s\n", errorFocusColor, file, lineNumber, resetColor))
	}

	for i := startLine; i <= endLine; i++ {
		lineColor := errorLineColor

		if i == line {
			lineColor = errorFocusColor
		}

		lineNumber := e.offset + i + 1
		b.WriteString(fmt.Sprintf("%s%3d%s %s\n", lineColor, lineNumber, resetColor, lines[i]))

		if i == line {
			length := len(lines[line])
			underline := strings.Repeat("^", length)
			b.WriteString(fmt.Sprintf("    %s%s%s\n", errorColor, underline, resetColor))
			b.WriteString(fmt.Sprintf("    %s%s%s\n", errorColor, e.msg, resetColor))
		}
	}

	return b.String()
}

func YamlParseError(err error, file string, src string) error {
	matches := yamlLineErrorRegex.FindStringSubmatch(err.Error())

	if len(matches) < 2 {
		return err
	}

	line, _ := strconv.Atoi(matches[1])
	msg := yamlLineErrorRegex.ReplaceAllString(err.Error(), "")
	return &codeFrameError{
		summary: "yaml parse error",
		file:    file,
		line:    line,
		src:     src,
		err:     err,
		msg:     msg,
	}
}

func TemplateParseError(err error, file string, src string, offset int) error {
	matches := templateParseErrorRegex.FindStringSubmatch(err.Error())

	if len(matches) < 2 {
		return err
	}

	line, _ := strconv.Atoi(matches[1])
	msg := templateParseErrorRegex.ReplaceAllString(err.Error(), "")
	return &codeFrameError{
		summary: "template parse error",
		file:    file,
		line:    line,
		offset:  offset,
		src:     src,
		err:     err,
		msg:     msg,
	}
}

func TemplateExecError(err error, file string, src string, offset int) error {
	matches := templateExecErrorRegex.FindStringSubmatch(err.Error())

	if len(matches) < 3 {
		return err
	}

	line, _ := strconv.Atoi(matches[1])
	column, _ := strconv.Atoi(matches[2])
	//name := matches[3]
	msg := templateExecErrorRegex.ReplaceAllString(err.Error(), "")
	return &codeFrameError{
		summary: "template evaluation error",
		file:    file,
		line:    line,
		offset:  offset,
		column:  column,
		src:     src,
		err:     err,
		msg:     msg,
	}
}

func ParseJsonError(err error, file string, src string) error {
	if jsonError, ok := err.(*json.SyntaxError); ok {
		line := strings.Count(src[:jsonError.Offset], "\n") + 1

		return &codeFrameError{
			summary: "json parse error",
			file:    file,
			line:    line,
			src:     src,
			err:     err,
			msg:     jsonError.Error(),
		}
	}

	if err, ok := err.(*json.UnmarshalTypeError); ok {
		line := strings.Count(src[:err.Offset], "\n") + 1

		return &codeFrameError{
			summary: "json invalid type",
			file:    file,
			line:    line,
			src:     src,
			err:     err,
			msg:     fmt.Sprintf("expected %s.%s to be a %s", err.Struct, err.Field, err.Type),
		}
	}

	return err
}

func FmtError(err error) string {
	cferr, ok := err.(*codeFrameError)

	if ok {
		return fmt.Sprintf("%serror:%s %s\n\n%s", errorColor, resetColor, cferr.summary, cferr)
	} else {
		return fmt.Sprintf("%serror:%s %s", errorColor, resetColor, err)
	}
}

var htmlColorCodes = map[string]string{
	errorColor:      `<span style="color: #e41010; font-weight: bold">`,
	errorLineColor:  `<span style="color: #adadad">`,
	errorFocusColor: `<span style="font-weight: bold">`,
	resetColor:      `</span>`,
}

func FmtErrorHtml(err error) string {
	str := FmtError(err)
	str = html.EscapeString(str)

	for color, tag := range htmlColorCodes {
		str = strings.ReplaceAll(str, color, tag)
	}

	var style strings.Builder
	style.WriteString("overflow-x: auto;")
	style.WriteString("font-family: Consolas,Menlo,Monaco,monospace;")
	style.WriteString("border-radius:8px;")
	style.WriteString("margin: 32px;")
	style.WriteString("border: solid 3px #e41010;")
	style.WriteString("padding: 16px;")
	return fmt.Sprintf(`<pre style="%s">%s</pre>`, style.String(), str)
}
