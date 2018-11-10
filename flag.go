package main

import (
	"github.com/kardianos/service"
	"github.com/urfave/cli"
	"strings"
)

func getApp() *cli.App {
	app := cli.NewApp()
	app.Name = "dohproxy"
	app.HelpName = app.Name
	app.Usage = "dohproxy -c config_file"
	app.Version = AppVer
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "show help",
		},
		cli.VersionFlag,
		cli.StringFlag{
			Name:  "config, c",
			Usage: "set config file",
			Value: "/etc/dohproxy.yml",
		},
		cli.StringFlag{
			Name:  "service, s",
			Usage: "service " + strings.Join(service.ControlAction[:], ","),
		},
		cli.BoolFlag{
			Name:   "from-service",
			Hidden: true,
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("help") {
			cli.ShowAppHelpAndExit(c, 0)
		}
		runApp(c)
		return nil
	}
	app.HideHelp = true
	return app
}
