---
str: page
num: 35
txt: |
  hello
list:
  - a
  - b
  - c
---

- str: {{ .Data.str }}
- num: {{ .Data.num }}
- txt: {{ .Data.txt }}

{{ range .Data.list }}
- {{ . }}
{{ end }}
