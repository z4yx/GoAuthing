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
	"gopkg.in/urfave/cli.v1"

	"github.com/z4yx/GoAuthing/libauth"
	"github.com/z4yx/GoAuthing/libtunet"
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
	AcID     string `json:"acId"`
	Campus   bool   `json:"campusOnly"`
}

var logger = loggo.GetLogger("auth-thu")
var settings Settings

func parseSettingsFile(path string) error {
	sf, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("read config file failed (%s)", err)
	}
	defer sf.Close()
	bv, _ := ioutil.ReadAll(sf)
	err = json.Unmarshal(bv, &settings)
	if err != nil {
		return fmt.Errorf("parse config file \"%s\" failed (%s)", path, err)
	}
	logger.Debugf("Read config file \"%s\" succeeded\n", path)
	return nil
}

func mergeCliSettings(c *cli.Context) {
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
	merged.AcID = c.String("ac-id")
	if len(merged.AcID) == 0 {
		merged.AcID = settings.AcID
	}
	merged.Campus = settings.Campus || c.Bool("campus-only")
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
	logger.Debugf("Settings AcID: \"%s\"\n", settings.AcID)
	logger.Debugf("Settings Campus: %t\n", settings.Campus)
}

func requestUser() (err error) {
	if len(settings.Username) == 0 && !settings.Daemon {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Username: ")
		settings.Username, _ = reader.ReadString('\n')
		settings.Username = strings.TrimSpace(settings.Username)
	}
	if len(settings.Username) == 0 {
		err = fmt.Errorf("username can't be empty")
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
			err = fmt.Errorf("interrupted")
			return
		}
		settings.Password = string(b)
	}
	if len(settings.Password) == 0 {
		err = fmt.Errorf("password can't be empty")
	}
	return
}

func setLoggerLevel(debug bool, daemon bool) {
	if daemon {
		_ = loggo.ConfigureLoggers("auth-thu=ERROR;libtunet=ERROR;libauth=ERROR")
	} else if debug {
		_ = loggo.ConfigureLoggers("auth-thu=DEBUG;libtunet=DEBUG;libauth=DEBUG")
	} else {
		_ = loggo.ConfigureLoggers("auth-thu=INFO;libtunet=INFO;libauth=INFO")
	}
}

func locateConfigFile(c *cli.Context) (cf string) {
	cf = c.GlobalString("config-file")
	if len(cf) != 0 {
		return
	}

	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	homedir, _ := os.UserHomeDir()
	if len(xdgConfigHome) == 0 {
		xdgConfigHome = path.Join(homedir, ".config")
	}
	cf = path.Join(xdgConfigHome, "auth-thu")
	_, err := os.Stat(cf)
	if !os.IsNotExist(err) {
		return
	}

	cf = path.Join(homedir, ".auth-thu")
	_, err = os.Stat(cf)
	if !os.IsNotExist(err) {
		return
	}

	return ""
}

func parseSettings(c *cli.Context) (err error) {
	if c.Bool("help") {
		cli.ShowAppHelpAndExit(c, 0)
	}
	// Early debug flag setting (have debug messages when access config file)
	setLoggerLevel(c.GlobalBool("debug"), c.GlobalBool("daemonize"))

	cf := locateConfigFile(c)
	if len(cf) == 0 && c.GlobalBool("daemonize") {
		return fmt.Errorf("cannot find config file (it is necessary in daemon mode)")
	}
	if len(cf) != 0 {
		err = parseSettingsFile(cf)
		if err != nil {
			return err
		}
	}
	mergeCliSettings(c)
	// Late debug flag setting
	setLoggerLevel(settings.Debug, settings.Daemon)
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
	logger.Infof("Accessing websites periodically to keep you online")

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
						FallbackDelay: -1,  // disable RFC 6555 Fast Fallback
					}
					return myDial.DialContext(ctx, network, addr)
				},
			},
		}
		resp, ret := netClient.Head(url)
		if ret != nil {
			return
		}
		defer resp.Body.Close()
		logger.Debugf("HTTP status code %d\n", resp.StatusCode)
		return
	}
	targetInside := "https://www.tsinghua.edu.cn/"
	targetOutside := "https://www.baidu.com/"

	stop := make(chan int, 1)
	defer func() { stop <- 1 }()
	go func() {
		// Keep IPv6 online, ignore any errors
		for {
			select {
			case <-stop:
				break
			case <-time.After(13 * time.Minute):
				_ = accessTarget(targetInside, true)
			}
		}
	}()

	for {
		target := targetOutside
		if campusOnly || settings.V6 {
			target = targetInside
		}
		if ret = accessTarget(target, settings.V6); ret != nil {
			ret = fmt.Errorf("accessing %s failed (re-login might be required): %w", target, ret)
			break
		}
		// Consumes ~5MB per day
		time.Sleep(3 * time.Second)
	}
	return
}

func authUtil(c *cli.Context, logout bool) error {
	err := parseSettings(c)
	if err != nil {
		return err
	}
	acID := "1"
	if len(settings.AcID) != 0 {
		acID = settings.AcID
	}
	domain := settings.Host
	if len(settings.Host) == 0 {
		if settings.V6 {
			domain = "auth6.tsinghua.edu.cn"
		} else {
			domain = "auth4.tsinghua.edu.cn"
		}
	}

	if len(settings.Ip) == 0 && len(settings.AcID) == 0 {
		// Probe the ac_id parameter
		// We do this only in Tsinghua, since it requires access to usereg.t.e.c/net.t.e.c
		// For v6, ac_id must be probed using different url
		retAcID, err := libauth.GetAcID(settings.V6)
		// FIXME: currently when logout, the GetAcID is actually broken.
		// Though logout does not require correct ac_id now, it can break.
		if err != nil && !logout {
			logger.Debugf("Failed to get ac_id: %v", err)
			logger.Debugf("Login may fail with 'IP地址异常'.")
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
			logger.Infof("Currently online!")
			if settings.KeepOn {
				return keepAliveLoop(c, true)
			}
			return nil
		} else if !online && logout {
			logger.Infof("Currently offline!")
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
		if len(settings.Ip) != 0 && len(settings.Host) == 0 && len(settings.AcID) == 0 {
			// Auth for another IP requires correct NAS ID since July 2020
			// Tsinghua only
			if retNasID, err := libauth.GetNasID(settings.Ip, settings.Username, settings.Password); err == nil {
				acID = retNasID
			}
		}
	}

	if settings.Campus {
		settings.Username += "@tsinghua"
	}

	err = libauth.LoginLogout(settings.Username, settings.Password, host, logout, settings.Ip, acID)
	action := "Login"
	if logout {
		action = "Logout"
	}
	if err == nil {
		logger.Infof("%s Successfully!\n", action)
		runHook(c)
		if settings.KeepOn {
			if len(settings.Ip) != 0 {
				logger.Errorf("Cannot keep another IP online\n")
			} else {
				return keepAliveLoop(c, true)
			}
		}
	} else {
		err = fmt.Errorf("%s Failed: %w", action, err)
	}
	return err
}

func cmdAuth(c *cli.Context) {
	logout := c.Bool("logout")
	err := authUtil(c, logout)
	if err != nil {
		logger.Errorf("Auth error: %s", err)
		os.Exit(1)
	}
}

func cmdDeauth(c *cli.Context) {
	err := authUtil(c, true)
	if err != nil {
		logger.Errorf("Deauth error: %s\n", err)
		os.Exit(1)
	}
}

func cmdLogin(c *cli.Context) error {
	err := parseSettings(c)
	if err != nil {
	    logger.Errorf("Parse setting error: %s\n", err)
	    os.Exit(1)
	}
	err = requestUser()
	if err != nil {
		logger.Errorf("Request user error: %s\n", err)
		os.Exit(1)
	}
	err = requestPasswd()
	if err != nil {
		logger.Errorf("Request password error: %s\n", err)
		os.Exit(1)
	}
	success, err := libtunet.LoginLogout(settings.Username, settings.Password, false)
	if success {
		logger.Infof("Login Successfully!\n")
		runHook(c)
		if settings.KeepOn {
			return keepAliveLoop(c, false)
		}
	} else {
		logger.Errorf("Login error: %s\n", err)
		os.Exit(1)
	}
	return err
}

func cmdLogout(c *cli.Context) {
	err := parseSettings(c)
	if err != nil {
		logger.Errorf("Parse setting error: %s\n", err)
		os.Exit(1)
	}
	//err := requestUser()
	success, err := libtunet.LoginLogout(settings.Username, settings.Password, true)
	if success {
		logger.Infof("Logout Successfully!\n")
		runHook(c)
	} else {
		logger.Errorf("Logout Failed: %s\n", err)
		os.Exit(1)
	}
}

func cmdKeepalive(c *cli.Context) {
	err := parseSettings(c)
	if err != nil {
		logger.Errorf("Parse setting error: %s\n", err)
		os.Exit(1)
	}
	err = keepAliveLoop(c, c.Bool("auth"))
	if err != nil {
	    logger.Errorf("Keepalive error: %s\n", err)
		os.Exit(1)
	}
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
		Version:  "2.2.1",
		HideHelp: true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username, u", Usage: "your TUNET account `name`"},
			&cli.StringFlag{Name: "password, p", Usage: "your TUNET `password`"},
			&cli.StringFlag{Name: "config-file, c", Usage: "`path` to your config file, default ~/.auth-thu"},
			&cli.StringFlag{Name: "hook-success", Usage: "command line to be executed in shell after successful login/out"},
			&cli.BoolFlag{Name: "daemonize, D", Usage: "run without reading username/password from standard input; less log"},
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
					&cli.StringFlag{Name: "ac-id", Usage: "use specified ac_id"},
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
					&cli.StringFlag{Name: "ac-id", Usage: "use specified ac_id"},
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
					&cli.BoolFlag{Name: "ipv6, 6", Usage: "keep only ipv6 connection online"},
				},
				Action: cmdKeepalive,
			},
		},
		Action: cmdAuth,
		Authors: []cli.Author{
			{Name: "Yuxiang Zhang", Email: "yuxiang.zhang@tuna.tsinghua.edu.cn"},
			{Name: "Nogeek", Email: "ritou11@gmail.com"},
			{Name: "ZenithalHourlyRate", Email: "i@zenithal.me"},
			{Name: "Jiajie Chen", Email: "c@jia.je"},
			{Name: "KomeijiOcean", Email: "oceans2000@126.com"},
			{Name: "Sharzy L", Email: "me@sharzy.in"},
		},
	}

	_ = app.Run(os.Args)
}
