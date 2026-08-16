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
	"when": func(ms uint64) string { return time.UnixMilli(int64(ms)).UTC().Format("2006-01-02 15:04:05") },
}

var tmpl = template.Must(template.New("").Funcs(funcs).Parse(`
{{define "layout_top"}}
<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>minichain explorer</title><style>
:root{color-scheme:dark;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace}*{box-sizing:border-box}body{min-height:100vh;margin:0;background:#07182b;color:#eff6ff}body::before{content:"";position:fixed;inset:0;z-index:-1;background:radial-gradient(circle at 15% 10%,#12365b 0,transparent 36rem),linear-gradient(135deg,#07182b,#0b2038)}header{border-bottom:1px solid #28425c;padding:1.8rem clamp(1.5rem,5vw,4rem)}.brand{color:#f8fbff;font-size:clamp(1.4rem,3vw,2rem);font-weight:800;letter-spacing:-.05em;text-decoration:none}main{max-width:100rem;margin:0 auto;padding:4rem clamp(1.5rem,5vw,4rem)}h1,h2,h3{letter-spacing:-.055em}h1{margin:0;font-family:ui-sans-serif,system-ui,sans-serif;font-size:clamp(2.5rem,6vw,4.5rem)}h2{font-family:ui-sans-serif,system-ui,sans-serif;font-size:2rem}a{color:#4de0d5}form{margin:0}input[type=text]{width:100%;border:1px solid #36516b;border-radius:.8rem;background:#0a1b2f;color:#eff6ff;padding:.95rem 1rem .95rem 3rem;font:inherit;outline:0}input[type=text]:focus{border-color:#45ddd0;box-shadow:0 0 0 3px #45ddd033}input[type=submit]{display:none}.search{position:relative;min-width:min(100%,28rem)}.search::before{content:"⌕";position:absolute;top:.48rem;left:1rem;color:#45ddd0;font-family:ui-sans-serif,sans-serif;font-size:1.8rem;line-height:1}.hero{display:flex;align-items:center;justify-content:space-between;gap:2rem;margin-bottom:3.5rem}.eyebrow{margin:.8rem 0 0;color:#93a9bd;font-size:.85rem;letter-spacing:.08em;text-transform:uppercase}.block-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(17rem,1fr));gap:1.5rem}.block-card{display:block;min-height:18rem;border:1px solid #2d4962;border-radius:.85rem;background:linear-gradient(145deg,#10283f,#0b1d31);color:inherit;padding:1.8rem;text-decoration:none;transition:transform .18s ease,border-color .18s ease}.block-card:hover{border-color:#45ddd0;transform:translateY(-3px)}.card-title{display:flex;align-items:center;gap:.8rem;margin:0 0 1.5rem;font-size:1.45rem}.cube{display:grid;width:2.3rem;height:2.3rem;place-items:center;border:2px solid #45ddd0;border-radius:.45rem;color:#45ddd0;font-size:1.2rem;transform:rotate(30deg)}.cube span{transform:rotate(-30deg)}.metric{padding:1rem 0;border-top:1px solid #28435b}.metric:first-of-type{border-top:0}.metric-label{display:block;margin-bottom:.45rem;color:#94aabd;font-family:ui-sans-serif,system-ui,sans-serif;font-size:.85rem}.metric-value{display:block;overflow:hidden;color:#f8fbff;text-overflow:ellipsis;white-space:nowrap}.valid{color:#45ddd0}table{width:100%;margin-top:1rem;border-collapse:collapse;background:#0d2238}th,td{padding:.75rem;border-bottom:1px solid #29435b;text-align:left}th{color:#a9bccd}.muted{color:#94aabd}@media(max-width:46rem){.hero{align-items:stretch;flex-direction:column}main{padding-top:2.5rem}}
</style></head><body><header><a class="brand" href="/">minichain explorer</a></header><main>{{end}}

{{define "index"}}{{template "layout_top" .}}<section class="hero"><div><h1>Latest blocks</h1><p class="eyebrow">height: {{.Height}} · proof-of-work chain</p></div><form class="search" action="/search" method="get"><input type="text" name="q" placeholder="Search block or address" aria-label="Search block or address"><input type="submit" value="search"></form></section><section class="block-grid">{{range .Blocks}}<a class="block-card" href="/block/{{.Index}}"><h2 class="card-title"><span class="cube"><span>◇</span></span>Block #{{.Index}}</h2><div class="metric"><span class="metric-label">Hash</span><span class="metric-value">{{short .Hash}}</span></div><div class="metric"><span class="metric-label">Transactions</span><span class="metric-value">{{len .Transactions}}</span></div><div class="metric"><span class="metric-label">Mined (UTC)</span><span class="metric-value">{{when .Timestamp}}</span></div><div class="metric"><span class="metric-label">Proof of work</span><span class="metric-value valid">Valid</span></div></a>{{end}}</section></main></body></html>{{end}}

{{define "block"}}{{template "layout_top" .}}<h2>block #{{.Block.Index}}</h2><table><tr><th>hash</th><td>{{.Block.Hash}}</td></tr><tr><th>prev</th><td><a href="/block/{{.Block.PrevHash}}">{{.Block.PrevHash}}</a></td></tr><tr><th>mined</th><td>{{when .Block.Timestamp}}</td></tr><tr><th>nonce</th><td>{{.Block.Nonce}}</td></tr></table><h3>transactions ({{len .Block.Transactions}})</h3><table><tr><th>from</th><th>to</th><th>amount</th><th>nonce</th></tr>{{range .Block.Transactions}}<tr><td>{{if eq .From "COINBASE"}}<span class="muted">coinbase</span>{{else}}<a href="/address/{{.From}}">{{short .From}}</a>{{end}}</td><td><a href="/address/{{.To}}">{{short .To}}</a></td><td>{{.Amount}}</td><td>{{if eq .From "COINBASE"}}<span class="muted">—</span>{{else}}{{.Nonce}}{{end}}</td></tr>{{end}}</table></main></body></html>{{end}}

{{define "address"}}{{template "layout_top" .}}<h2>address {{short .Address}}</h2><p>{{.Address}}</p><p>balance: <strong>{{.Balance}}</strong></p><h3>history ({{len .History}})</h3><table><tr><th>block</th><th>from</th><th>to</th><th>amount</th></tr>{{range .History}}<tr><td><a href="/block/{{.BlockIndex}}">{{.BlockIndex}}</a></td><td>{{if eq .From "COINBASE"}}<span class="muted">coinbase</span>{{else}}<a href="/address/{{.From}}">{{short .From}}</a>{{end}}</td><td><a href="/address/{{.To}}">{{short .To}}</a></td><td>{{.Amount}}</td></tr>{{end}}</table></main></body></html>{{end}}

{{define "notfound"}}{{template "layout_top" .}}<p>nothing found for <strong>{{.Query}}</strong></p></main></body></html>{{end}}
`))
