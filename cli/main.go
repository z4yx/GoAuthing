package main

import (
	"fmt"
	"os"

	"github.com/juju/loggo"
	"github.com/z4yx/GoAuthing/libauth"
	cli "gopkg.in/urfave/cli.v2"
)

var logger = loggo.GetLogger("")

func cmdAction(c *cli.Context) error {
	if c.Bool("debug") {
		loggo.ConfigureLoggers("<root>=DEBUG;libauth=DEBUG")
	} else {
		loggo.ConfigureLoggers("<root>=INFO;libauth=INFO")
	}
	username := c.String("username")
	password := c.String("password")
	success, err := libauth.Login(username, password, 4)
	if success {
		fmt.Printf("Login Successfully!\n")
	} else {
		fmt.Printf("Login Failed: %s\n", err.Error())
	}
	return nil
}

func main() {
	app := &cli.App{
		Name:      "auth-thu",
		UsageText: "auth-thu [-u <username>] [-p <password>] [options]",
		Usage:     "Authenticating utility for auth.tsinghua.edu.cn",
		Version:   "0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Aliases: []string{"u"}, Usage: "your TUNET account name"},
			&cli.StringFlag{Name: "password", Aliases: []string{"p"}, Usage: "your TUNET password"},
			&cli.BoolFlag{Name: "debug", Usage: "print debug messages"},
		},
		Action:  cmdAction,
		Authors: []*cli.Author{{Name: "Yuxiang Zhang", Email: "yuxiang.zhang@tuna.tsinghua.edu.cn"}},
	}

	app.Run(os.Args)
}
