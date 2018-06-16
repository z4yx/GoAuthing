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
	logout := c.Bool("logout")
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
		if online && !logout {
			fmt.Println("Currently online!")
			return nil
		} else if !online && logout {
			fmt.Println("Currently offline!")
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
	if len(username) == 0 {
		return cli.NewExitError("username can't be empty", 1)
	}

	password := c.String("password")
	if !logout && len(password) == 0 {
		fmt.Printf("Password: ")
		b, err := gopass.GetPasswdMasked()
		if err != nil {
			// Handle gopass.ErrInterrupted or getch() read error
			return nil
		}
		password = string(b)
	}
	if !logout && len(password) == 0 {
		return cli.NewExitError("password can't be empty", 1)
	}

	success, err := libauth.LoginLogout(username, password, proto, logout)
	action := "Login"
	if logout {
		action = "Logout"
	}
	if success {
		fmt.Printf("%s Successfully!\n", action)
	} else {
		fmt.Printf("%s Failed: %s\n", action, err.Error())
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
			&cli.BoolFlag{Name: "logout, o", Usage: "log out of the online account"},
			&cli.BoolFlag{Name: "ipv6, 6", Usage: "authenticating for IPv6 (auth6)"},
			&cli.BoolFlag{Name: "help, h", Usage: "print the help"},
			&cli.BoolFlag{Name: "debug", Usage: "print debug messages"},
		},
		Action:  cmdAction,
		Authors: []cli.Author{{Name: "Yuxiang Zhang", Email: "yuxiang.zhang@tuna.tsinghua.edu.cn"}},
	}

	app.Run(os.Args)
}
