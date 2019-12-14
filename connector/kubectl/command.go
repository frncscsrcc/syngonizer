package kubectl

import (
	"os/exec"

	"github.com/frncscsrcc/syngonizer/log"
)

type semaphore chan bool

var sem semaphore

func initializeCommandLimiter(limit int) {
	sem = make(semaphore, limit)
}

// acquire n resources
func (s semaphore) Increment(n int) {
	e := true
	for i := 0; i < n; i++ {
		s <- e
	}
}

// release n resources
func (s semaphore) Decrement(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}

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

func execCommands(log log.Log, commands ...*command) {
	// Use semaphore to be sure we are not running to much go routine
	// potentialy causing an auto DoS
	sem.Increment(1)
	defer sem.Decrement(1)

	for _, c := range commands {
		out, err := c.exec()
		// Igore the error, if it is required
		if c.ignoreErr == false && err != nil {
			log.SendError(err)
			return
		}
		// Ignore output, if it is required
		if c.isSilent {
			continue
		}
		if out != "" {
			log.SendDebug(out)
		}
	}
}

func execCommandsBackground(log log.Log, commands ...*command) {
	go execCommands(log, commands...)
}
