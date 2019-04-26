package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/juju/loggo"
	cli "gopkg.in/urfave/cli.v1"

	"auth-thu/libauth"
	"auth-thu/libtunet"
)

type Settings struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Ip       string `json:"ip"`
	Host     string `json:"host"`
	NoCheck  bool   `json:"noCheck"`
	V6       bool   `json:"useV6"`
	Insecure bool   `json:"insecure"`
	Debug    bool   `json:"debug"`
}

var logger = loggo.GetLogger("")
var settings Settings

func parseSettingsFile(path string) error {
	sf, err := os.Open(path)
	if err != nil {
		return err
	}
	defer sf.Close()
	bv, _ := ioutil.ReadAll(sf)
	json.Unmarshal(bv, &settings)
	return nil
}

func mergeCliSettings(c *cli.Context) error {
	var merged Settings
	merged.Username = c.GlobalString("username")
	if len(merged.Username) == 0 {
		merged.Username = settings.Username
	}
	merged.Password = c.GlobalString("password")
	if len(merged.Password) == 0 {
		merged.Password = settings.Password
	}
	merged.Ip = c.String("ip")
	if len(merged.Ip) == 0 {
		merged.Ip = settings.Ip
	}
	merged.Host = c.String("host")
	if len(merged.Host) == 0 {
		merged.Host = settings.Host
	}
	merged.NoCheck = settings.NoCheck || c.Bool("no-check")
	merged.V6 = settings.V6 || c.Bool("ipv6")
	merged.Insecure = settings.Insecure || c.Bool("insecure")
	merged.Debug = settings.Debug || c.GlobalBool("debug")
	settings = merged
	return nil
}

func requestUser() (err error) {
	if len(settings.Username) == 0 {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Username: ")
		settings.Username, _ = reader.ReadString('\n')
		settings.Username = strings.TrimSpace(settings.Username)
	}
	if len(settings.Username) == 0 {
		err = cli.NewExitError("username can't be empty", 1)
	}
	return
}

func requestPasswd() (err error) {
	if len(settings.Password) == 0 {
		var b []byte
		fmt.Printf("Password: ")
		b, err = gopass.GetPasswdMasked()
		if err != nil {
			// Handle gopass.ErrInterrupted or getch() read error
			err = cli.NewExitError("interrupted", 1)
			return
		}
		settings.Password = string(b)
	}
	if len(settings.Password) == 0 {
		err = cli.NewExitError("password can't be empty", 1)
	}
	return
}

func parseSettings(c *cli.Context) error {
	if c.Bool("help") {
		cli.ShowAppHelpAndExit(c, 0)
	}
	cf := c.GlobalString("config-file")
	if len(cf) == 0 {
		homedir, _ := os.UserHomeDir()
		cf = path.Join(homedir, ".auth-thu")
	}
	parseSettingsFile(cf)
	mergeCliSettings(c)
	return nil
}

func cmdAuth(c *cli.Context) error {
	parseSettings(c)
	logout := c.Bool("logout")
	acID := "1"
	if settings.Debug {
		loggo.ConfigureLoggers("<root>=DEBUG;libauth=DEBUG")
	} else {
		loggo.ConfigureLoggers("<root>=INFO;libauth=INFO")
	}
	domain := settings.Host
	if len(domain) == 0 {
		if settings.V6 {
			domain = "auth6.tsinghua.edu.cn"
		} else {
			domain = "auth4.tsinghua.edu.cn"
		}

		// Probe the ac_id parameter
		// We do this only in Tsinghua, since it requires access to net.tsinghua.edu.cn
		if retAcID, err := libauth.GetAcID(); err == nil {
			acID = retAcID
		}
	}
	host := libauth.NewUrlProvider(domain, settings.Insecure)
	if len(settings.Ip) == 0 && !settings.NoCheck {
		online, _ := libauth.IsOnline(host, acID)
		if online && !logout {
			fmt.Println("Currently online!")
			return nil
		} else if !online && logout {
			fmt.Println("Currently offline!")
			return nil
		}
	}
	err := requestUser()
	if err != nil {
		return err
	}
	if !logout {
		err = requestPasswd()
		if err != nil {
			return err
		}
	}

	success, err := libauth.LoginLogout(settings.Username, settings.Password, host, logout, settings.Ip, acID)
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

func cmdLogin(c *cli.Context) error {
	parseSettings(c)
	if settings.Debug {
		loggo.ConfigureLoggers("<root>=DEBUG;libtunet=DEBUG")
	} else {
		loggo.ConfigureLoggers("<root>=INFO;libtunet=INFO")
	}
	err := requestUser()
	if err != nil {
		return err
	}
	err = requestPasswd()
	if err != nil {
		return err
	}
	success, err := libtunet.LoginLogout(settings.Username, settings.Password, false)
	if success {
		fmt.Printf("Login Successfully!\n")
	} else {
		fmt.Printf("Login Failed: %s\n", err.Error())
	}
	return err
}

func cmdLogout(c *cli.Context) error {
	parseSettings(c)
	if settings.Debug {
		loggo.ConfigureLoggers("<root>=DEBUG;libtunet=DEBUG")
	} else {
		loggo.ConfigureLoggers("<root>=INFO;libtunet=INFO")
	}
	err := requestUser()
	if err != nil {
		return err
	}
	success, err := libtunet.LoginLogout(settings.Username, settings.Password, true)
	if success {
		fmt.Printf("Logout Successfully!\n")
	} else {
		fmt.Printf("Logout Failed: %s\n", err.Error())
	}
	return err
}

func main() {
	app := &cli.App{
		Name: "auth-thu",
		UsageText: `auth-thu [options]
	 auth-thu [options] auth [auth_options]
	 auth-thu [options] login
	 auth-thu [options] logout`,
		Usage:    "Authenticating utility for Tsinghua",
		Version:  "1.3.2",
		HideHelp: true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username, u", Usage: "your TUNET account `name`"},
			&cli.StringFlag{Name: "password, p", Usage: "your TUNET `password`"},
			&cli.StringFlag{Name: "config-file, c", Usage: "`path` to your config file, default ~/.auth-thu"},
			&cli.BoolFlag{Name: "debug", Usage: "print debug messages"},
			&cli.BoolFlag{Name: "help, h", Usage: "print the help"},
		},
		Commands: []cli.Command{
			cli.Command{
				Name:  "auth",
				Usage: "(default) Auth via auth4/6.tsinghua",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "ip", Usage: "authenticating for specified IP address"},
					&cli.BoolFlag{Name: "no-check, n", Usage: "skip online checking, always send login request"},
					&cli.BoolFlag{Name: "logout, o", Usage: "log out of the online account"},
					&cli.BoolFlag{Name: "ipv6, 6", Usage: "authenticating for IPv6 (auth6.tsinghua)"},
					&cli.StringFlag{Name: "host", Usage: "use customized hostname of srun4000"},
					&cli.BoolFlag{Name: "insecure", Usage: "use http instead of https"},
				},
				Action: cmdAuth,
			},
			cli.Command{
				Name:   "login",
				Usage:  "Login via net.tsinghua",
				Action: cmdLogin,
			},
			cli.Command{
				Name:   "logout",
				Usage:  "Logout via net.tsinghua",
				Action: cmdLogout,
			},
		},
		Action: cmdAuth,
		Authors: []cli.Author{
			{Name: "Yuxiang Zhang", Email: "yuxiang.zhang@tuna.tsinghua.edu.cn"},
			{Name: "Nogeek", Email: "ritou11@gmail.com"},
		},
	}

	app.Run(os.Args)
}
