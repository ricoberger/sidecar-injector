package version

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrint(t *testing.T) {
	goVersion := runtime.Version()

	Version = "v0.4.0"
	Revision = "cc373f263575773f1349bbd354e803cc85f9edcd"
	Branch = "main"
	BuildUser = "root"
	BuildDate = "2022-05-03@20:00:00"

	version, err := Print("sidecar-injector")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("sidecar-injector, version v0.4.0 (branch: main, revision: cc373f263575773f1349bbd354e803cc85f9edcd)\n  build user:       root\n  build date:       2022-05-03@20:00:00\n  go version:       %s", goVersion), version)
}

func TestInfo(t *testing.T) {
	Version = "v0.4.0"
	Revision = "cc373f263575773f1349bbd354e803cc85f9edcd"
	Branch = "main"

	fields := Info()
	require.Equal(t, []any{"version", "v0.4.0", "branch", "main", "revision", "cc373f263575773f1349bbd354e803cc85f9edcd"}, fields)
}

func TestBuildContext(t *testing.T) {
	goVersion := runtime.Version()

	BuildUser = "root"
	BuildDate = "2022-05-03@20:00:00"

	fields := BuildContext()
	require.Equal(t, []any{"go", goVersion, "user", "root", "date", "2022-05-03@20:00:00"}, fields)
}
