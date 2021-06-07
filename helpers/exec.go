package helpers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/osixia/container-baseimage/log"
)

// Exec wrapper
// =============================

type Exec struct {
	context context.Context
	timeout time.Duration
	setGPID bool

	Cmd     *exec.Cmd
	pidFile string

	stdout *bytes.Buffer
}

func NewExec(ctx context.Context) *Exec {

	return &Exec{
		context: ctx,
	}
}

func (e *Exec) WithTimeout(timeout time.Duration) *Exec {
	e.timeout = timeout

	return e
}

func (e *Exec) WithStdout(b *bytes.Buffer) *Exec {
	e.stdout = b

	return e
}

func (e *Exec) WithSetGPID(b bool) *Exec {
	e.setGPID = b

	return e
}

func (e *Exec) WithPIDFile(f string) *Exec {
	e.pidFile = f

	return e
}

func (e *Exec) Command(name string, args ...string) *Exec {
	log.Tracef("Exec.Command called with cmd: %v, args: %v", name, args)

	cmd := exec.CommandContext(e.context, name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	if e.stdout != nil {
		cmd.Stdout = e.stdout
	}

	if e.setGPID {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	}

	e.Cmd = cmd

	return e
}

func (e *Exec) Run() error {

	if err := e.Start(); err != nil {
		return err
	}

	return e.Wait()
}

func (e *Exec) Start() error {

	log.Infof("Running %v ...", e.Cmd)

	if err := e.Cmd.Start(); err != nil {
		return err
	}

	pid := e.Cmd.Process.Pid

	if e.pidFile != "" {
		f, err := Create(e.pidFile)
		if err != nil {
			log.Warningf("error creating pid file %v: %v", e.pidFile, err.Error())
		} else {
			if _, err := f.WriteString(strconv.Itoa(pid)); err != nil {
				log.Warningf("error writing in pid file %v: %v", e.pidFile, err.Error())
			}
		}
	}

	log.Debugf("%v: started (pid %v)", e.Cmd, pid)

	return nil
}

func (e *Exec) Wait() error {

	var err error
	if e.timeout > 0 {
		err = e.waitOrStop(e.context, e.Cmd, os.Interrupt, e.timeout)
	} else {
		err = e.Cmd.Wait()
	}

	if e.pidFile != "" {
		if err := Remove(e.pidFile); err != nil {
			log.Warningf("error removing pid file %v: %v", e.pidFile, err.Error())
		}
	}

	if errors.Is(err, context.Canceled) {
		err = nil
	}

	if err != nil {
		err = fmt.Errorf("%v: %w", e.Cmd, err)
	}

	return err
}

// waitOrStop waits for the already-started command cmd by calling its Wait method.
//
// If cmd does not return before ctx is done, waitOrStop sends it the given interrupt signal.
// If killDelay is positive, waitOrStop waits that additional period for Wait to return before sending os.Kill.
//
// This function is copied from the one added to x/playground/internal in
// http://golang.org/cl/228438.
func (e *Exec) waitOrStop(ctx context.Context, cmd *exec.Cmd, interrupt os.Signal, killDelay time.Duration) error {

	log.Tracef("Exec.waitOrStop called with cmd: %v, interrupt: %v, killDelay: %v", cmd, interrupt, killDelay)

	if cmd.Process == nil {
		log.Fatal("waitOrStop called with a nil cmd.Process — missing Start call?")
	}
	if interrupt == nil {
		log.Fatal("waitOrStop requires a non-nil interrupt signal")
	}

	errc := make(chan error)
	go func() {
		select {
		case errc <- nil:
			return
		case <-ctx.Done():
		}

		err := cmd.Process.Signal(interrupt)
		if err == nil {
			err = ctx.Err() // Report ctx.Err() as the reason we interrupted.
		} else if errors.Is(err, os.ErrProcessDone) {
			errc <- nil
			return
		}

		if killDelay > 0 {
			timer := time.NewTimer(killDelay)
			select {
			// Report ctx.Err() as the reason we interrupted the process...
			case errc <- ctx.Err():
				timer.Stop()
				return
			// ...but after killDelay has elapsed, fall back to a stronger signal.
			case <-timer.C:
			}

			// Wait still hasn't returned.
			// Kill the process harder to make sure that it exits.
			//
			// Ignore any error: if cmd.Process has already terminated, we still
			// want to send ctx.Err() (or the error from the Interrupt call)
			// to properly attribute the signal that may have terminated it.
			_ = cmd.Process.Kill()
		}

		errc <- err
	}()

	waitErr := cmd.Wait()
	if interruptErr := <-errc; interruptErr != nil {
		return interruptErr
	}
	return waitErr
}
