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

func requestUser(c *cli.Context) (username string, err error) {
	username = c.String("username")
	if len(username) == 0 {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Username: ")
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
	}
	if len(username) == 0 {
		err = cli.NewExitError("username can't be empty", 1)
	}
	return
}

func requestPasswd(c *cli.Context) (password string, err error) {
	password = c.String("password")
	if len(password) == 0 {
		var b []byte
		fmt.Printf("Password: ")
		b, err = gopass.GetPasswdMasked()
		if err != nil {
			// Handle gopass.ErrInterrupted or getch() read error
			err = cli.NewExitError("interrupted", 1)
			return
		}
		password = string(b)
	}
	if len(password) == 0 {
		err = cli.NewExitError("password can't be empty", 1)
	}
	return
}

func cmdAction(c *cli.Context) error {
	logout := c.Bool("logout")
	anotherIP := c.String("ip")
	if c.Bool("help") {
		cli.ShowAppHelpAndExit(c, 0)
	}
	if c.Bool("debug") {
		loggo.ConfigureLoggers("<root>=DEBUG;libauth=DEBUG")
	} else {
		loggo.ConfigureLoggers("<root>=INFO;libauth=INFO")
	}
	domain := c.String("host")
	if len(domain) == 0 {
		if c.Bool("ipv6") {
			domain = "auth6.tsinghua.edu.cn"
		} else {
			domain = "auth4.tsinghua.edu.cn"
		}
	}
	host := libauth.NewUrlProvider(domain, c.Bool("insecure"))
	if len(anotherIP) == 0 && !c.Bool("no-check") {
		online, _ := libauth.IsOnline(host)
		if online && !logout {
			fmt.Println("Currently online!")
			return nil
		} else if !online && logout {
			fmt.Println("Currently offline!")
			return nil
		}
	}
	username, err := requestUser(c)
	if err != nil {
		return err
	}
	password := ""
	if !logout {
		password, err = requestPasswd(c)
		if err != nil {
			return err
		}
	}

	success, err := libauth.LoginLogout(username, password, host, logout, anotherIP)
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
		Usage:     "Authenticating utility for auth.tsinghua.edu.cn (srun4000)",
		Version:   "0.3.1",
		HideHelp:  true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username, u", Usage: "your TUNET account `name`"},
			&cli.StringFlag{Name: "password, p", Usage: "your TUNET `password`"},
			&cli.StringFlag{Name: "ip", Usage: "authenticating for specified IP address"},
			&cli.BoolFlag{Name: "no-check, n", Usage: "skip online checking, always send login request"},
			&cli.BoolFlag{Name: "logout, o", Usage: "log out of the online account"},
			&cli.BoolFlag{Name: "ipv6, 6", Usage: "authenticating for IPv6 (auth6.tsinghua)"},
			&cli.StringFlag{Name: "host", Usage: "use customized hostname of srun4000"},
			&cli.BoolFlag{Name: "insecure", Usage: "use http instead of https"},
			&cli.BoolFlag{Name: "help, h", Usage: "print the help"},
			&cli.BoolFlag{Name: "debug", Usage: "print debug messages"},
		},
		Action:  cmdAction,
		Authors: []cli.Author{{Name: "Yuxiang Zhang", Email: "yuxiang.zhang@tuna.tsinghua.edu.cn"}},
	}

	app.Run(os.Args)
}
