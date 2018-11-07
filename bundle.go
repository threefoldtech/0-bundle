package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codegangsta/cli"
)

type Bundle struct {
	chroot  *Chroot
	sandbox *Sandbox
}

func (bundle *Bundle) Run(ctx *cli.Context, updateCh chan bool) {
	defer bundle.chroot.Stop()
	signalChan := make(chan os.Signal)
	//listen for termination signals
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	for {
		log.Debugf("starting chroot")
		if err := bundle.chroot.Start(); err != nil {
			log.Error(err.Error())
			return
		}
		exitChan := make(chan struct{}, 1)
		go bundle.execSandbox(ctx, exitChan)

		select {
		case <-updateCh:
			log.Info("Flist updates were found, trying to restart 0-bundle ...")
			sandboxTerminated := bundle.stopSandbox(exitChan)
			bundle.chroot.Stop()
			bundle.chroot.Wait()
			close(exitChan)
			if sandboxTerminated {
				continue
			}
			log.Error("Failed to stop zbundle sandbox, exiting...")
			return
		case <-exitChan:
			if ctx.GlobalBool("no-exit") {
				bundle.sandBoxNoExit(signalChan)
			}
			return
		case sigCh := <-signalChan:
			bundle.sandbox.Signal(sigCh)
			if ctx.GlobalBool("no-exit") {
				bundle.sandBoxNoExit(signalChan)
			}
			return
		}
	}
	return
}

func (bundle *Bundle) stopSandbox(exitChan chan struct{}) bool {
	err := bundle.sandbox.Signal(syscall.SIGTERM)
	if err != nil {
		return true
	}
	//retry to terminate sandbox 3 times
	for i := 0; i < 3; i++ {
		select {
		case <-exitChan:
			return true
		case <-time.After(5 * time.Second):
			log.Infof("Failed to stop bundle sandbox, retry %d/3", i+1)
			bundle.sandbox.Signal(syscall.SIGTERM)
		}
	}
	bundle.sandbox.Signal(syscall.SIGKILL)
	return false
}

func (bundle *Bundle) execSandbox(ctx *cli.Context, exitChan chan struct{}) {
	stdout, stderr, err := bundle.sandbox.Run()
	if err != nil {
		if err := report(ctx, stdout, stderr, err); err != nil {
			log.Errorf("report: %s", err)
		}
	}
	exitChan <- struct{}{}
}

func (bundle *Bundle) sandBoxNoExit(ch chan os.Signal) {
	log.Infof("flist exited, waiting for unmount (--no-exit was set)")
	log.Infof("the sandbox is mounted under: %s", bundle.chroot.MountRoot())
	log.Infof("Ctrl+C to terminate the sandbox")
	go func() {
		//wait for termination signal to terminate the sandbox
		<-ch
		log.Infof("terminating ...")
		bundle.chroot.Stop()
	}()
	bundle.chroot.Wait()
}
