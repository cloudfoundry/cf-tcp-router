package haproxy_test

import (
	. "code.cloudfoundry.org/cf-tcp-router/configurer/haproxy"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("CommandRunner", func() {
	var (
		cmdRunner *CommandRunner
		logger    lager.Logger
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("script-runner-test")
	})
	Describe("Run", func() {
		Context("when the underlying script exits successfully", func() {
			BeforeEach(func() {
				cmdRunner = CreateCommandRunner("fixtures/testscript", logger)
			})
			It("runs script successfully", func() {
				err := cmdRunner.Run(false)
				Expect(err).ToNot(HaveOccurred())
				Expect(logger).Should(gbytes.Say("hello test"))
			})
			It("logs a useful message", func() {
				cmdRunner.Run(false)
				logs := logger.(*lagertest.TestLogger).Logs()
				Expect(len(logs)).To(Equal(1))
				Expect(logs[0].Message).To(Equal("script-runner-test.running-script"))
				Expect(logs[0].Data).To(Equal(lager.Data{
					"command": "fixtures/testscript",
					"output":  "hello test\nIS_DRAINING=\n",
					"error":   nil,
				}))
			})
			Context("when called with forceHealthCheckToFail set to false", func() {
				It("launches the runnerCmd without setting IS_DRAINING=true", func() {
					err := cmdRunner.Run(false)
					Expect(err).NotTo(HaveOccurred())
					Expect(logger).ToNot(gbytes.Say("setting-drain-mode"))
					Expect(logger).ToNot(gbytes.Say("IS_DRAINING=true"))

				})
			})
			Context("when called with forceHealthCheckToFail set to true", func() {
				It("launches the runnerCmd with IS_DRAINING=true", func() {
					err := cmdRunner.Run(true)
					Expect(err).NotTo(HaveOccurred())
					Expect(logger).To(gbytes.Say("setting-drain-mode"))
					Expect(logger).To(gbytes.Say("IS_DRAINING=true"))

				})
			})
		})

		Context("when the underlying script does not exist", func() {
			BeforeEach(func() {
				cmdRunner = CreateCommandRunner("fixtures/non-existent-script", logger)
			})
			It("throws error", func() {
				err := cmdRunner.Run(false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no such file or directory"))
			})
		})

		Context("when the underlying script errors", func() {
			BeforeEach(func() {
				cmdRunner = CreateCommandRunner("fixtures/badscript", logger)
			})
			It("throws error", func() {
				err := cmdRunner.Run(false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("exit status 1"))
				Expect(logger).Should(gbytes.Say("negative test"))
			})
		})
	})
})
