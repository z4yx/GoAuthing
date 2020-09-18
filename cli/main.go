package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

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
	HookSucc string `json:"hook-success"`
	NoCheck  bool   `json:"noCheck"`
	KeepOn   bool   `json:"keepOnline"`
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
	merged.HookSucc = c.GlobalString("hook-success")
	if len(merged.HookSucc) == 0 {
		merged.HookSucc = settings.HookSucc
	}
	merged.NoCheck = settings.NoCheck || c.Bool("no-check")
	merged.V6 = settings.V6 || c.Bool("ipv6")
	merged.KeepOn = settings.KeepOn || c.Bool("keep-online")
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

func runHook(c *cli.Context) {
	if settings.HookSucc != "" {
		logger.Debugf("Run hook \"%s\"\n", settings.HookSucc)
		cmd := exec.Command(settings.HookSucc)
		if err := cmd.Run(); err != nil {
			logger.Errorf("Hook execution failed: %v\n", err)
		}
	}
}

func keepAliveLoop(c *cli.Context, campusOnly bool) (ret error) {
	fmt.Println("Accessing websites periodically to keep you online")

	accessTarget := func(url string, ipv6 bool) (ret error) {
		network := "tcp4"
		if ipv6 {
			network = "tcp6"
		}
		netClient := &http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _network, addr string) (net.Conn, error) {
					logger.Debugf("DialContext %s (%s)\n", addr, network)
					myDial := &net.Dialer{
						Timeout:   6 * time.Second,
						KeepAlive: 0,
						DualStack: false,
					}
					return myDial.DialContext(ctx, network, addr)
				},
			},
		}
		for errorCount := 0; errorCount < 3; errorCount++ {
			var resp *http.Response
			resp, ret = netClient.Get(url)
			if ret == nil {
				defer resp.Body.Close()
				logger.Debugf("HTTP status code %d\n", resp.StatusCode)
				_, ret := ioutil.ReadAll(resp.Body)
				if ret == nil {
					break
				}
			}
		}
		return
	}
	targetInside := "https://www.tsinghua.edu.cn/"
	targetOutside := "http://www.baidu.com/"

	stop := make(chan int, 1)
	defer func() { stop <- 1 }()
	go func() {
		// Keep IPv6 online, ignore any errors
		for {
			select {
			case <-stop:
				break
			case <-time.After(13 * time.Minute):
				accessTarget(targetInside, true)
			}
		}
	}()

	v4Target := targetOutside
	if campusOnly {
		v4Target = targetInside
	}
	for {
		if ret = accessTarget(v4Target, false); ret != nil {
			logger.Errorf("Accessing %s: %v\n", v4Target, ret)
			fmt.Printf("Failed to access %s, you have to re-login.\n", v4Target)
			break
		}
		// Consumes ~100MB per day
		time.Sleep(55 * time.Second)
	}
	return
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
	if len(settings.Host) == 0 {
		if settings.V6 {
			domain = "auth6.tsinghua.edu.cn"
		} else {
			domain = "auth4.tsinghua.edu.cn"
		}

		if len(settings.Ip) == 0 {
			// Probe the ac_id parameter
			// We do this only in Tsinghua, since it requires access to usereg.t.e.c/net.t.e.c
			// For v6, ac_id must be probed using different url
			if retAcID, err := libauth.GetAcID(settings.V6); err == nil {
				acID = retAcID
			}
		}
	}
	host := libauth.NewUrlProvider(domain, settings.Insecure)
	if len(settings.Ip) == 0 && !settings.NoCheck {
		online, _, username := libauth.IsOnline(host, acID)
		settings.Username = username
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
		if len(settings.Ip) != 0 && len(settings.Host) == 0 {
			// Auth for another IP requires correct NAS ID since July 2020
			// Tsinghua only
			if retNasID, err := libauth.GetNasID(settings.Ip, settings.Username, settings.Password); err == nil {
				acID = retNasID
			}
		}
	}

	success, err := libauth.LoginLogout(settings.Username, settings.Password, host, logout, settings.Ip, acID)
	action := "Login"
	if logout {
		action = "Logout"
	}
	if success {
		fmt.Printf("%s Successfully!\n", action)
		runHook(c)
		if settings.KeepOn {
			if len(settings.Ip) != 0 {
				fmt.Printf("Cannot keep another IP online\n")
			} else {
				return keepAliveLoop(c, true)
			}
		}
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
		runHook(c)
		if settings.KeepOn {
			return keepAliveLoop(c, false)
		}
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
	//err := requestUser()
	//if err != nil {
	//	return err
	//}
	success, err := libtunet.LoginLogout(settings.Username, settings.Password, true)
	if success {
		fmt.Printf("Logout Successfully!\n")
		runHook(c)
	} else {
		fmt.Printf("Logout Failed: %s\n", err.Error())
	}
	return err
}

func cmdKeepalive(c *cli.Context) error {
	parseSettings(c)
	if settings.Debug {
		loggo.ConfigureLoggers("<root>=DEBUG")
	} else {
		loggo.ConfigureLoggers("<root>=INFO")
	}
	return keepAliveLoop(c, c.Bool("auth"))
}

func main() {
	app := &cli.App{
		Name: "auth-thu",
		UsageText: `auth-thu [options]
	 auth-thu [options] auth [auth_options]
	 auth-thu [options] login
	 auth-thu [options] logout`,
		Usage:    "Authenticating utility for Tsinghua",
		Version:  "1.7",
		HideHelp: true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username, u", Usage: "your TUNET account `name`"},
			&cli.StringFlag{Name: "password, p", Usage: "your TUNET `password`"},
			&cli.StringFlag{Name: "config-file, c", Usage: "`path` to your config file, default ~/.auth-thu"},
			&cli.StringFlag{Name: "hook-success", Usage: "command line to be executed in shell after successful login/out"},
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
					&cli.BoolFlag{Name: "keep-online, k", Usage: "keep online after login"},
				},
				Action: cmdAuth,
			},
			cli.Command{
				Name:  "login",
				Usage: "Login via net.tsinghua",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "keep-online, k", Usage: "keep online after login"},
				},
				Action: cmdLogin,
			},
			cli.Command{
				Name:   "logout",
				Usage:  "Logout via net.tsinghua",
				Action: cmdLogout,
			},
			cli.Command{
				Name:  "online",
				Usage: "Keep your computer online",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "auth, a", Usage: "keep the Auth online only"},
				},
				Action: cmdKeepalive,
			},
		},
		Action: cmdAuth,
		Authors: []cli.Author{
			{Name: "Yuxiang Zhang", Email: "yuxiang.zhang@tuna.tsinghua.edu.cn"},
			{Name: "Nogeek", Email: "ritou11@gmail.com"},
			{Name: "ZenithalHourlyRate", Email: "i@zenithal.me"},
		},
	}

	app.Run(os.Args)
}
