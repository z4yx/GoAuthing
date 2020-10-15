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
	Daemon   bool   `json:"daemonize"`
	Debug    bool   `json:"debug"`
	AcID     *int   `json:"acId"`
}

var logger = loggo.GetLogger("auth-thu")
var settings Settings

func parseSettingsFile(path string, important bool) error {
	sf, err := os.Open(path)
	if err != nil {
		if important {
			logger.Errorf("Read config file \"%s\" failed (may be existence or access problem)\n", path)
			err = cli.NewExitError("Read config file failed", 1)
			return err
		} else {
			logger.Debugf("Read config file \"%s\" failed (may be existence or access problem)\n", path)
			return nil
		}
	}
	logger.Debugf("Read config file \"%s\" succeeded\n", path)
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
	merged.Daemon = settings.Daemon || c.GlobalBool("daemonize")
	merged.Debug = settings.Debug || c.GlobalBool("debug")
	merged.AcID = settings.AcID
	if c.IsSet("ac-id") {
		// command line flag takes precedence
		var acID int = c.Int("ac-id")
		merged.AcID = &acID
	}
	settings = merged
	logger.Debugf("Settings Username: \"%s\"\n", settings.Username)
	logger.Debugf("Settings Ip: \"%s\"\n", settings.Ip)
	logger.Debugf("Settings Host: \"%s\"\n", settings.Host)
	logger.Debugf("Settings HookSucc: \"%s\"\n", settings.HookSucc)
	logger.Debugf("Settings NoCheck: %t\n", settings.NoCheck)
	logger.Debugf("Settings V6: %t\n", settings.V6)
	logger.Debugf("Settings KeepOn: %t\n", settings.KeepOn)
	logger.Debugf("Settings Insecure: %t\n", settings.Insecure)
	logger.Debugf("Settings Daemon: %t\n", settings.Daemon)
	logger.Debugf("Settings Debug: %t\n", settings.Debug)
	if settings.AcID != nil {
		logger.Debugf("Settings AcID: %d\n", *settings.AcID)
	} else {
		logger.Debugf("Settings AcID: unspecified\n")
	}
	return nil
}

func requestUser() (err error) {
	if len(settings.Username) == 0 && !settings.Daemon {
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
	if len(settings.Password) == 0 && !settings.Daemon {
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

func setLoggerLevel(debug bool) {
	if debug {
		loggo.ConfigureLoggers("auth-thu=DEBUG;libtunet=DEBUG;libauth=DEBUG")
	} else {
		loggo.ConfigureLoggers("auth-thu=INFO;libtunet=INFO;libauth=INFO")
	}
}

func parseSettings(c *cli.Context) (err error) {
	if c.Bool("help") {
		cli.ShowAppHelpAndExit(c, 0)
	}
	// Early debug flag setting (have debug messages when access config file)
	setLoggerLevel(c.GlobalBool("debug"))
	cf := c.GlobalString("config-file")
	if len(cf) == 0 {
		homedir, _ := os.UserHomeDir()
		cf = path.Join(homedir, ".auth-thu")
		// If run in daemon mode, config file is a must
		err = parseSettingsFile(cf, c.GlobalBool("daemonize"))
	} else {
		err = parseSettingsFile(cf, true)
	}
	if err != nil {
		return
	}
	mergeCliSettings(c)
	// Late debug flag setting
	setLoggerLevel(settings.Debug)
	return
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

func cmdAuthUtil(c *cli.Context, logout bool) error {
	err := parseSettings(c)
	if err != nil {
		return err
	}
	acID := "1"
	if settings.AcID != nil {
		acID = fmt.Sprint(*settings.AcID)
	}
	domain := settings.Host
	if len(settings.Host) == 0 {
		if settings.V6 {
			domain = "auth6.tsinghua.edu.cn"
		} else {
			domain = "auth4.tsinghua.edu.cn"
		}
	}

	if len(settings.Ip) == 0 && settings.AcID == nil {
		// Probe the ac_id parameter
		// We do this only in Tsinghua, since it requires access to usereg.t.e.c/net.t.e.c
		// For v6, ac_id must be probed using different url
		retAcID, err := libauth.GetAcID(settings.V6)
		// FIXME: currently when logout, the GetAcID is actually broken.
		// Though logout does not require correct ac_id now, it can break.
		if err != nil && !logout {
			logger.Warningf("Failed to get ac_id: %v", err)
			logger.Warningf("Login may fail with 'IP地址异常'.")
		}
		acID = retAcID
	}

	host := libauth.NewUrlProvider(domain, settings.Insecure)
	if len(settings.Ip) == 0 && !settings.NoCheck {
		online, _, username := libauth.IsOnline(host, acID)
		if logout && online {
			settings.Username = username
		}
		if online && !logout {
			fmt.Println("Currently online!")
			return nil
		} else if !online && logout {
			fmt.Println("Currently offline!")
			return nil
		}
	}
	err = requestUser()
	if err != nil {
		return err
	}
	if !logout {
		err = requestPasswd()
		if err != nil {
			return err
		}
		if len(settings.Ip) != 0 && len(settings.Host) == 0 && settings.AcID == nil {
			// Auth for another IP requires correct NAS ID since July 2020
			// Tsinghua only
			if retNasID, err := libauth.GetNasID(settings.Ip, settings.Username, settings.Password); err == nil {
				acID = retNasID
			}
		}
	}

	if c.Bool("campus-only") {
		settings.Username += "@tsinghua"
	}

	err = libauth.LoginLogout(settings.Username, settings.Password, host, logout, settings.Ip, acID)
	action := "Login"
	if logout {
		action = "Logout"
	}
	if err == nil {
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
		fmt.Printf("%s Failed: %v\n", action, err)
	}
	return err
}

func cmdAuth(c *cli.Context) error {
	logout := c.Bool("logout")
	return cmdAuthUtil(c, logout)
}

func cmdDeauth(c *cli.Context) error {
	return cmdAuthUtil(c, true)
}

func cmdLogin(c *cli.Context) error {
	err := parseSettings(c)
	if err != nil {
		return err
	}
	err = requestUser()
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
	err := parseSettings(c)
	if err != nil {
		return err
	}
	//err := requestUser()
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
	err := parseSettings(c)
	if err != nil {
		return err
	}
	return keepAliveLoop(c, c.Bool("auth"))
}

func main() {
	app := &cli.App{
		Name: "auth-thu",
		UsageText: `auth-thu [options]
	 auth-thu [options] auth [auth_options]
	 auth-thu [options] deauth [auth_options]
	 auth-thu [options] login
	 auth-thu [options] logout
	 auth-thu [options] online [online_options]`,
		Usage:    "Authenticating utility for Tsinghua",
		Version:  "1.9.7",
		HideHelp: true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username, u", Usage: "your TUNET account `name`"},
			&cli.StringFlag{Name: "password, p", Usage: "your TUNET `password`"},
			&cli.StringFlag{Name: "config-file, c", Usage: "`path` to your config file, default ~/.auth-thu"},
			&cli.StringFlag{Name: "hook-success", Usage: "command line to be executed in shell after successful login/out"},
			&cli.BoolFlag{Name: "daemonize, D", Usage: "run without reading username/password from standard input"},
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
					&cli.BoolFlag{Name: "logout, o", Usage: "de-auth of the online account (behaves the same as deauth command, for backward-compatibility)"},
					&cli.BoolFlag{Name: "ipv6, 6", Usage: "authenticating for IPv6 (auth6.tsinghua)"},
					&cli.BoolFlag{Name: "campus-only, C", Usage: "auth only, no auto-login (v4 only)"},
					&cli.StringFlag{Name: "host", Usage: "use customized hostname of srun4000"},
					&cli.BoolFlag{Name: "insecure", Usage: "use http instead of https"},
					&cli.BoolFlag{Name: "keep-online, k", Usage: "keep online after login"},
					&cli.IntFlag{Name: "ac-id", Usage: "use specified ac_id"},
				},
				Action: cmdAuth,
			},
			cli.Command{
				Name:  "deauth",
				Usage: "De-auth via auth4/6.tsinghua",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "ip", Usage: "authenticating for specified IP address"},
					&cli.BoolFlag{Name: "no-check, n", Usage: "skip online checking, always send logout request"},
					&cli.BoolFlag{Name: "ipv6, 6", Usage: "authenticating for IPv6 (auth6.tsinghua)"},
					&cli.StringFlag{Name: "host", Usage: "use customized hostname of srun4000"},
					&cli.BoolFlag{Name: "insecure", Usage: "use http instead of https"},
					&cli.IntFlag{Name: "ac-id", Usage: "use specified ac_id"},
				},
				Action: cmdDeauth,
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
