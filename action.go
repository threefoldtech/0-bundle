package main

import (
	"fmt"
	"net/url"

	"github.com/codegangsta/cli"
)

var (
//BaseFSDir where we keep the cache and the working place of fuse
//BaseFSDir    = path.Join(os.TempDir(), "zbundle.db")
//BaseMountDir = path.Join(os.TempDir(), "zbundle")
)

func action(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("flist is missing")
	}

	if !isRoot() {
		return fmt.Errorf("please run as root")
	}

	updateChan := make(chan bool)
	// Check for flist url and start updateChecker routine
	flist := ctx.Args().First()
	_, err := url.Parse(flist)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	flistHash, err := getFlistHash(flist)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	updateInterval := ctx.GlobalInt("update")
	go checkFlistUpdate(flist, flistHash, updateInterval, updateChan)

	// Start the sandbox
	chroot := Chroot{
		ID:      ctx.GlobalString("id"),
		Flist:   ctx.Args().First(),
		Storage: ctx.GlobalString("storage"),
	}

	sandbox := Sandbox{
		Root:       chroot.MountRoot(),
		UserEnv:    ctx.GlobalStringSlice("env"),
		EntryPoint: ctx.GlobalString("entry-point"),
		Args:       ctx.Args().Tail(),
	}
	bundle := Bundle{
		chroot:  &chroot,
		sandbox: &sandbox,
	}
	bundle.Run(ctx, updateChan)

	return nil
}
