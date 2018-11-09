package main

import "github.com/urfave/cli"

func getApp() *cli.App {
	app := cli.NewApp()
	app.Name = "dohproxy"
	app.HelpName = app.Name
	app.Usage = "dohproxy -c config_file]"
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
