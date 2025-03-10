package version

import (
	"fmt"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVersion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Version Suite")
}

var _ = Describe("Version", func() {
	Context("Version Information", func() {
		It("Should print version information", func() {
			goVersion := runtime.Version()
			Version = "v0.4.0"
			Revision = "cc373f263575773f1349bbd354e803cc85f9edcd"
			Branch = "main"
			BuildUser = "root"
			BuildDate = "2022-05-03@20:00:00"

			version, err := Print("sidecar-injector")

			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(fmt.Sprintf("sidecar-injector, version v0.4.0 (branch: main, revision: cc373f263575773f1349bbd354e803cc85f9edcd)\n  build user:       root\n  build date:       2022-05-03@20:00:00\n  go version:       %s", goVersion)))
		})

		It("Should return info", func() {
			Version = "v0.4.0"
			Revision = "cc373f263575773f1349bbd354e803cc85f9edcd"
			Branch = "main"

			fields := Info()

			Expect(fields).To(Equal([]any{"version", "v0.4.0", "branch", "main", "revision", "cc373f263575773f1349bbd354e803cc85f9edcd"}))
		})

		It("Should return build context", func() {
			goVersion := runtime.Version()
			BuildUser = "root"
			BuildDate = "2022-05-03@20:00:00"

			fields := BuildContext()

			Expect(fields).To(Equal([]any{"go", goVersion, "user", "root", "date", "2022-05-03@20:00:00"}))
		})
	})
})
