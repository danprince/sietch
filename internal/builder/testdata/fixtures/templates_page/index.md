Explicit: {{ with page "./a.md" }}{{ .Data.title }}{{ end }}

Implicit: {{ with page "./b/" }}{{ .Data.title }}{{ end }}
