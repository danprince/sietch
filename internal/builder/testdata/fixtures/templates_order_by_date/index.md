Ascending
{{ range index | orderByDate "asc" }}
<p>{{ .Data.title }} {{ .Date.Format "2006-1-2" }}</p>
{{ end }}

---

Descending
{{ range index | orderByDate "desc" }}
<p>{{ .Data.title }} {{ .Date.Format "2006-1-2" }}</p>
{{ end }}
