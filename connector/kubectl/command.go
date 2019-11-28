package kubectl

import "os/exec"

type command struct {
	cmd       string
	args      []string
	ignoreErr bool
	isSilent  bool
}

func newCommand(cmd string, args ...string) *command {
	c := command{
		cmd:       cmd,
		args:      args,
		ignoreErr: false,
		isSilent:  false,
	}
	return &c
}

func (c *command) ignoreErrors(flag bool) *command {
	c.ignoreErr = flag
	return c
}

func (c *command) beSilent(flag bool) *command {
	c.isSilent = flag
	return c
}

func (c *command) exec() (string, error) {
	out, err := exec.Command(c.cmd, c.args...).CombinedOutput()
	return string(out), err
}

func execCommands(errorChan chan error, commands ...*command) (string, error) {
	output := ""
	for _, c := range commands {
		out, err := c.exec()
		// Igore the error, if it is required
		if c.ignoreErr != true && err != nil {
			errorChan <- err
			return "", err
		}
		// Ignore output, if it is requred
		if c.isSilent {
			continue
		}
		if out != "" {
			errorChan <- err
		}
		output += out
	}
	return output, nil
}

func backgroundExecCommands(errorChan chan error, commands ...*command) {
	go execCommands(errorChan, commands...)
}
