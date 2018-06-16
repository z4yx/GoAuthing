package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/juju/loggo"
	"github.com/z4yx/GoAuthing/libauth"
	cli "gopkg.in/urfave/cli.v1"
)

var logger = loggo.GetLogger("")

func cmdAction(c *cli.Context) error {
	proto := 4
	if c.Bool("help") {
		cli.ShowAppHelpAndExit(c, 0)
	}
	if c.Bool("debug") {
		loggo.ConfigureLoggers("<root>=DEBUG;libauth=DEBUG")
	} else {
		loggo.ConfigureLoggers("<root>=INFO;libauth=INFO")
	}
	if c.Bool("ipv6") {
		proto = 6
	}
	if !c.Bool("no-check") {
		online, _ := libauth.IsOnline(proto)
		if online {
			fmt.Println("Currently online!")
			return nil
		}
	}
	username := c.String("username")
	if len(username) == 0 {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Username: ")
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
	}
	password := c.String("password")
	if len(password) == 0 {
		fmt.Printf("Password: ")
		b, err := gopass.GetPasswdMasked()
		if err != nil {
			// Handle gopass.ErrInterrupted or getch() read error
			return nil
		}
		password = string(b)
	}
	if len(username) == 0 || len(password) == 0 {
		return cli.NewExitError("username or password can't be empty", 1)
	}
	success, err := libauth.Login(username, password, proto)
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
		HideHelp:  true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username, u", Usage: "your TUNET account `name`"},
			&cli.StringFlag{Name: "password, p", Usage: "your TUNET `password`"},
			&cli.BoolFlag{Name: "no-check, n", Usage: "skip online checking, always send login request"},
			&cli.BoolFlag{Name: "ipv6, 6", Usage: "authenticating for IPv6 (auth6)"},
			&cli.BoolFlag{Name: "help, h", Usage: "print the help"},
			&cli.BoolFlag{Name: "debug", Usage: "print debug messages"},
		},
		Action:  cmdAction,
		Authors: []cli.Author{{Name: "Yuxiang Zhang", Email: "yuxiang.zhang@tuna.tsinghua.edu.cn"}},
	}

	app.Run(os.Args)
}
