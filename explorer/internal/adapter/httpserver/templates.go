package httpserver

import (
	"html/template"
	"time"
)

var funcs = template.FuncMap{
	"short": func(s string) string {
		if len(s) <= 16 {
			return s
		}
		return s[:8] + "…" + s[len(s)-8:]
	},
	"when": func(ms uint64) string {
		return time.UnixMilli(int64(ms)).UTC().Format("2006-01-02 15:04:05")
	},
}

var tmpl = template.Must(template.New("").Funcs(funcs).Parse(`
{{define "layout_top"}}
<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>minichain explorer</title>
<style>
  body { font-family: ui-monospace, monospace; margin: 2rem auto; max-width: 60rem; color: #222; }
  h1 a { color: inherit; text-decoration: none; }
  table { border-collapse: collapse; width: 100%; margin-top: 1rem; }
  th, td { text-align: left; padding: .4rem .6rem; border-bottom: 1px solid #ddd; }
  th { background: #f5f5f5; }
  form { margin: 1rem 0; }
  input[type=text] { width: 30rem; padding: .4rem; font-family: inherit; }
  a { color: #0366d6; }
  .muted { color: #888; }
</style>
</head>
<body>
<h1><a href="/">minichain explorer</a></h1>
<form action="/search" method="get">
  <input type="text" name="q" placeholder="block index, block hash or address">
  <input type="submit" value="search">
</form>
{{end}}

{{define "index"}}
{{template "layout_top" .}}
<p>height: {{.Height}}</p>
<table>
<tr><th>#</th><th>hash</th><th>mined (UTC)</th><th>txs</th></tr>
{{range .Blocks}}
<tr>
  <td><a href="/block/{{.Index}}">{{.Index}}</a></td>
  <td><a href="/block/{{.Hash}}">{{short .Hash}}</a></td>
  <td>{{when .Timestamp}}</td>
  <td>{{len .Transactions}}</td>
</tr>
{{end}}
</table>
</body></html>
{{end}}

{{define "block"}}
{{template "layout_top" .}}
<h2>block #{{.Block.Index}}</h2>
<table>
<tr><th>hash</th><td>{{.Block.Hash}}</td></tr>
<tr><th>prev</th><td><a href="/block/{{.Block.PrevHash}}">{{.Block.PrevHash}}</a></td></tr>
<tr><th>mined</th><td>{{when .Block.Timestamp}}</td></tr>
<tr><th>nonce</th><td>{{.Block.Nonce}}</td></tr>
</table>
<h3>transactions ({{len .Block.Transactions}})</h3>
<table>
<tr><th>from</th><th>to</th><th>amount</th><th>nonce</th></tr>
{{range .Block.Transactions}}
<tr>
  <td>{{if eq .From "COINBASE"}}<span class="muted">coinbase</span>{{else}}<a href="/address/{{.From}}">{{short .From}}</a>{{end}}</td>
  <td><a href="/address/{{.To}}">{{short .To}}</a></td>
  <td>{{.Amount}}</td>
  <td>{{if eq .From "COINBASE"}}<span class="muted">—</span>{{else}}{{.Nonce}}{{end}}</td>
</tr>
{{end}}
</table>
</body></html>
{{end}}

{{define "address"}}
{{template "layout_top" .}}
<h2>address {{short .Address}}</h2>
<p>{{.Address}}</p>
<p>balance: <strong>{{.Balance}}</strong></p>
<h3>history ({{len .History}})</h3>
<table>
<tr><th>block</th><th>from</th><th>to</th><th>amount</th></tr>
{{range .History}}
<tr>
  <td><a href="/block/{{.BlockIndex}}">{{.BlockIndex}}</a></td>
  <td>{{if eq .From "COINBASE"}}<span class="muted">coinbase</span>{{else}}<a href="/address/{{.From}}">{{short .From}}</a>{{end}}</td>
  <td><a href="/address/{{.To}}">{{short .To}}</a></td>
  <td>{{.Amount}}</td>
</tr>
{{end}}
</table>
</body></html>
{{end}}

{{define "notfound"}}
{{template "layout_top" .}}
<p>nothing found for <strong>{{.Query}}</strong></p>
</body></html>
{{end}}
`))
