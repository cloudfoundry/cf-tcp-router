package haproxy

import (
	"os/exec"

	"code.cloudfoundry.org/lager/v3"
)

//go:generate counterfeiter -o fakes/fake_script_runner.go . ScriptRunner
type ScriptRunner interface {
	Run(forceHealthCheckToFail bool) error
}

type CommandRunner struct {
	scriptPath string
	logger     lager.Logger
}

func CreateCommandRunner(scriptPath string, logger lager.Logger) *CommandRunner {
	return &CommandRunner{
		scriptPath: scriptPath,
		logger:     logger,
	}
}

func (cmd *CommandRunner) Run(forceHealthCheckToFail bool) error {
	runnerCmd := exec.Command(cmd.scriptPath)

	if forceHealthCheckToFail {
		cmd.logger.Debug("setting-drain-mode")
		runnerCmd.Env = append(runnerCmd.Env, "IS_DRAINING=true")
	}

	output, err := runnerCmd.CombinedOutput()
	cmd.logger.Info("running-script", lager.Data{"command": string(cmd.scriptPath), "output": string(output), "error": err})
	return err
}
