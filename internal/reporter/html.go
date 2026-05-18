package reporter

import (
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/JunaDev/cortixlabs/pkg/types"
)

// WriteHTML renders a self-contained HTML report to w and returns the suggested filename.
func WriteHTML(w io.Writer, result *types.ScanResult) (filename string, err error) {
	data := buildTemplateData(result)
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"upper":    strings.ToUpper,
		"severityClass": func(s types.Severity) string {
			switch s {
			case types.SeverityCritical:
				return "critical"
			case types.SeverityWarning:
				return "warning"
			case types.SeverityPass:
				return "pass"
			default:
				return "improvement"
			}
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("reporter: html template parse: %w", err)
	}

	if err := tmpl.Execute(w, data); err != nil {
		return "", fmt.Errorf("reporter: html template execute: %w", err)
	}

	filename = fmt.Sprintf("cortix-report-%s-%s.html",
		strings.ReplaceAll(result.ClusterName, "/", "-"),
		result.ScannedAt.Format("2006-01-02"))
	return filename, nil
}

type templateData struct {
	ClusterName string
	ScannedAt   time.Time
	Mode        types.ScanMode
	Score       *types.Score
	Categories  []types.CategoryResult
	Found       []types.DetectedTool
	Missing     []types.DetectedTool
	HealthScore int
}

func buildTemplateData(result *types.ScanResult) templateData {
	return templateData{
		ClusterName: result.ClusterName,
		ScannedAt:   result.ScannedAt,
		Mode:        result.Mode,
		Score:       result.Score,
		Categories:  result.Categories,
		Found:       result.Found,
		Missing:     result.Missing,
		HealthScore: result.HealthScore,
	}
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Cortix Scan Report — {{.ClusterName}}</title>
<style>
  :root { --red:#ef4444;--yellow:#f59e0b;--green:#22c55e;--blue:#3b82f6;--bg:#0f172a;--card:#1e293b;--text:#e2e8f0;--muted:#94a3b8; }
  * { box-sizing:border-box; margin:0; padding:0; }
  body { background:var(--bg); color:var(--text); font-family:'Segoe UI',system-ui,sans-serif; padding:2rem; }
  h1 { font-size:1.5rem; margin-bottom:.25rem; }
  .meta { color:var(--muted); font-size:.875rem; margin-bottom:2rem; }
  .score-card { background:var(--card); border-radius:.75rem; padding:1.5rem; margin-bottom:1.5rem; display:flex; align-items:center; gap:2rem; }
  .score-big { font-size:3.5rem; font-weight:700; }
  .score-verdict { color:var(--muted); font-size:1rem; margin-top:.25rem; }
  .category { background:var(--card); border-radius:.75rem; padding:1.25rem; margin-bottom:1rem; }
  .category-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:1rem; font-weight:600; }
  .check { padding:.5rem .75rem; border-radius:.375rem; margin-bottom:.375rem; font-size:.875rem; }
  .check.critical { background:rgba(239,68,68,.1); border-left:3px solid var(--red); }
  .check.warning  { background:rgba(245,158,11,.1); border-left:3px solid var(--yellow); }
  .check.pass     { background:rgba(34,197,94,.1);  border-left:3px solid var(--green); }
  .check.improvement { background:rgba(59,130,246,.1); border-left:3px solid var(--blue); }
  .remediation { color:var(--muted); font-size:.8rem; margin-top:.25rem; font-family:monospace; }
  .cta { text-align:center; margin-top:2rem; padding:1.5rem; background:var(--card); border-radius:.75rem; }
  .cta a { color:var(--blue); text-decoration:none; font-weight:600; }
  .badge { display:inline-block; padding:.125rem .5rem; border-radius:9999px; font-size:.75rem; font-weight:600; }
  .badge.critical { background:var(--red); }
  .badge.warning  { background:var(--yellow); color:#000; }
  .badge.pass     { background:var(--green); color:#000; }
  .delta-pos { color:var(--green); }
  .delta-neg { color:var(--red); }
</style>
</head>
<body>
<h1>Cortix Scan Report</h1>
<p class="meta">Cluster: <strong>{{.ClusterName}}</strong> &nbsp;·&nbsp; Scanned: {{.ScannedAt.Format "2006-01-02 15:04 UTC"}} &nbsp;·&nbsp; Mode: {{.Mode}}</p>

{{if .Score}}
<div class="score-card">
  <div>
    <div class="score-big">{{.Score.Overall}}<span style="font-size:1.5rem;color:var(--muted)">/100</span></div>
    <div class="score-verdict">{{.Score.Verdict}}</div>
  </div>
  <div style="color:var(--muted);font-size:.875rem;">
    Industry avg: {{.Score.IndustryAvg}}/100 &nbsp;
    {{if ge .Score.Delta 0}}<span class="delta-pos">▲ +{{.Score.Delta}}</span>{{else}}<span class="delta-neg">▼ {{.Score.Delta}}</span>{{end}}
  </div>
</div>
{{else if .HealthScore}}
<div class="score-card">
  <div class="score-big">{{.HealthScore}}<span style="font-size:1.5rem;color:var(--muted)">/10</span></div>
  <div class="score-verdict">Quick scan health score</div>
</div>
{{end}}

{{range .Categories}}
<div class="category">
  <div class="category-header">
    <span>{{upper (string .Category)}}</span>
    <span>{{.Score}}/100
      {{if ge (sub .Score .IndustryAvg) 0}}<span class="delta-pos">▲</span>{{else}}<span class="delta-neg">▼</span>{{end}}
      vs avg {{.IndustryAvg}}
    </span>
  </div>
  {{range .Checks}}
  <div class="check {{severityClass .Severity}}">
    <div><strong>{{.Name}}</strong>{{if .Resource}} — <code>{{.Resource}}</code>{{end}}</div>
    {{if .Detail}}<div style="color:var(--muted);margin-top:.2rem">{{.Detail}}</div>{{end}}
    {{if .Remediation}}<div class="remediation">→ {{.Remediation}}</div>{{end}}
  </div>
  {{end}}
</div>
{{end}}

{{if .Found}}
<div class="category">
  <div class="category-header"><span>OBSERVABILITY TOOLS</span></div>
  {{range .Found}}<div class="check pass"><strong>{{.Name}}</strong>{{if .Version}} v{{.Version}}{{end}} — {{.Namespace}}</div>{{end}}
  {{range .Missing}}<div class="check critical"><strong>{{.Name}}</strong> — missing</div>{{end}}
</div>
{{end}}

<div class="cta">
  Run <code>cortix install</code> or <a href="https://cortixlabs.io" target="_blank">visit cortixlabs.io</a> to fix this automatically.
</div>
</body>
</html>
`
