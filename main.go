package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("zbundle")
)

func init() {
	logging.SetLevel(logging.INFO, "")
}

func main() {
	// 1- mount the flist
	// 2- we need to export some env variables
	// 3- how to run ? bash, etc.
	app := cli.NewApp()

	app.Name = "zbundle"
	app.Usage = "run mini environments from flist"
	app.UsageText = "zbundle [options] <flist> <args>"
	app.Version = "1.0"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "id, i",
			Usage: "[required] ID of the sandbox",
		},
		cli.StringFlag{
			Name:  "entry-point",
			Value: "/etc/start",
			Usage: "sandbox entry point",
		},
		cli.StringSliceFlag{
			Name:  "env, e",
			Usage: "custom environemt variables",
		},
		cli.StringSliceFlag{
			Name:  "report, r",
			Usage: "report error back on failures to the given url",
		},
		cli.StringFlag{
			Name:  "storage, s",
			Value: "zdb://hub.grid.tf:9900",
			Usage: "storage url to use",
		},
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "run in debug mode",
		},
		cli.BoolFlag{
			Name:  "no-exit",
			Usage: "do not terminate (unmount the sandbox) after /etc/start exits",
		},
		cli.IntFlag{
			Name:  "update",
			Value: 30,
			Usage: "Check for flist updates interval in minutes",
		},
	}

	app.ArgsUsage = "flist"
	app.Before = func(ctx *cli.Context) error {
		logging.Reset()
		backend := logging.NewLogBackend(os.Stdout, "", 0)
		formatter := logging.MustStringFormatter("%{color}%{time:15:04:05.000} %{module} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}")
		logging.SetBackend(logging.NewBackendFormatter(backend, formatter))

		if ctx.GlobalBool("debug") {
			logging.SetLevel(logging.DEBUG, "")
		} else {
			logging.SetLevel(logging.INFO, "")
		}

		//validate required inputs
		for _, key := range []string{"id"} {
			value := ctx.GlobalString(key)
			if value == "" {
				return fmt.Errorf("flag --%s is required", key)
			}
		}

		return nil
	}
	app.Action = action

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
