{{ range index | orderByDate }}
<p>{{ .Data.title }} {{ .Date.Format "2006-1-2" }}</p>
{{ end }}
