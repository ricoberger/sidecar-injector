package version

import (
	"bytes"
	"runtime"
	"strings"
	"text/template"
)

// Build information. Populated at build-time.
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
	GoVersion = runtime.Version()
)

// versionInfoTmpl contains the template used by Print.
var versionInfoTmpl = `
{{.program}}, version {{.version}} (branch: {{.branch}}, revision: {{.revision}})
  build user:       {{.buildUser}}
  build date:       {{.buildDate}}
  go version:       {{.goVersion}}
`

// Print returns version information.
func Print(program string) (string, error) {
	data := map[string]string{
		"program":   program,
		"version":   Version,
		"revision":  Revision,
		"branch":    Branch,
		"buildUser": BuildUser,
		"buildDate": BuildDate,
		"goVersion": GoVersion,
	}

	var buf bytes.Buffer
	tmpl := template.Must(template.New("version").Parse(versionInfoTmpl))
	tmpl.ExecuteTemplate(&buf, "version", data)

	return strings.TrimSpace(buf.String()), nil
}

// Info returns version, branch and revision information.
func Info() []any {
	return []any{"version", Version, "branch", Branch, "revision", Revision}
}

// BuildContext returns goVersion, buildUser and buildDate information.
func BuildContext() []any {
	return []any{"go", GoVersion, "user", BuildUser, "date", BuildDate}
}
