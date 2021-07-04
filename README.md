# GoAuthing

[![Build Status](https://travis-ci.org/z4yx/GoAuthing.svg?branch=master)](https://travis-ci.org/z4yx/GoAuthing)
![GPLv3](https://img.shields.io/badge/license-GPLv3-blue.svg)

A commandline Tunet (auth4/6.tsinghua.edu.cn, Tsinghua-IPv4) authentication tool.

## Download Binary

Download prebuilt binaries from https://github.com/z4yx/GoAuthing/releases  
Or https://mirrors.tuna.tsinghua.edu.cn/github-release/z4yx/GoAuthing/

## Usage

Simply try `./auth-thu`, then enter your user name and password.

```
NAME:
   auth-thu - Authenticating utility for Tsinghua

USAGE:
   auth-thu [options]
   auth-thu [options] auth [auth_options]
   auth-thu [options] deauth [auth_options]
   auth-thu [options] login
   auth-thu [options] logout
   auth-thu [options] online [online_options]

VERSION:
   2.0.2

AUTHORS:
   Yuxiang Zhang <yuxiang.zhang@tuna.tsinghua.edu.cn>
   Nogeek <ritou11@gmail.com>
   ZenithalHourlyRate <i@zenithal.me>
   Jiajie Chen <c@jia.je>
   KomeijiOcean <oceans2000@126.com>
   Sharzy L <me@sharzy.in>

COMMANDS:
     auth    (default) Auth via auth4/6.tsinghua
       OPTIONS:
         --ip value         authenticating for specified IP address
         --no-check, -n     skip online checking, always send login request
         --logout, -o       de-auth of the online account (behaves the same as deauth command, for backward-compatibility)
         --ipv6, -6         authenticating for IPv6 (auth6.tsinghua)
         --campus-only, -C  auth only, no auto-login (v4 only)
         --host value       use customized hostname of srun4000
         --insecure         use http instead of https
         --keep-online, -k  keep online after login
         --ac-id value      use specified ac_id
     deauth  De-auth via auth4/6.tsinghua
       OPTIONS:
         --ip value      authenticating for specified IP address
         --no-check, -n  skip online checking, always send logout request
         --ipv6, -6      authenticating for IPv6 (auth6.tsinghua)
         --host value    use customized hostname of srun4000
         --insecure      use http instead of https
         --ac-id value   use specified ac_id
     login   Login via net.tsinghua
     logout  Logout via net.tsinghua
     online  Keep your computer online
       OPTIONS:
         --auth, -a  keep the Auth online only

GLOBAL OPTIONS:
   --username name, -u name          your TUNET account name
   --password password, -p password  your TUNET password
   --config-file path, -c path       path to your config file, default ~/.auth-thu
   --hook-success value              command line to be executed in shell after successful login/out
   --daemonize, -D                   run without reading username/password from standard input; less log
   --debug                           print debug messages
   --help, -h                        print the help
   --version, -v                     print the version
```

Write a config file to store your username & password or other options in the following format.
The default location of config file is `~/.auth-thu`.

```
{
  "username": "your-username",
  "password": "your-password",
  "host": "",
  "ip": "166.xxx.xx.xx",
  "debug": false,
  "useV6": false,
  "noCheck": false,
  "insecure": false,
  "daemonize": false,
  "acId": "",
  "campusOnly": false
}
```

Unless you have special need, you can only have `username` and `password` field in your config file. For `host`, the default value defined in code should be sufficient hence there should be no need to fill it. `UseV6` automatically determine the `host` to use. For `ip`, unless you are auth/login the other boxes you have(not the box `auth-thu` is running on), you can leave it blank. For those boxes unable to get correct acid themselves, we can specify the acid for them by using `acId`. Other options are self-explanatory.

To configure automatic authentication on systemd based Linux distro, take a look at `docs` folder. Just modify the path in configure files, then copy them to `/etc/systemd/system` folder.

Note that the program should have access to the configure file. For `goauthing.service`, since it is run as `nobody`, `/etc/goauthing.json` can not be read by it, hence you can use the following command to enable access.

```
setfacl -m u:nobody:r /etc/goauthing.json
```

Or, to be more secure, you can choose `goauthing@.service` and store the config in `~/.auth-thu`.

For other authentication like IPv6, you can copy these service files and modify them correspondingly.

It is suggested that one configures and runs it manually first with `debug` flag turned on, which ensures the correctness of one's config, then start it as system service. For `daemonize` flag, it forces the program to only log errors, hence debugging should be done earlier and manually. `daemonize` is automatically turned on for system service (ref to associated systemd unit files).

## Build

Requires Go 1.11 or above

```
export GO111MODULE=on
go build -o auth-thu github.com/z4yx/GoAuthing/cli
```

## Acknowledgments

This project was inspired by the following projects:

- https://github.com/jiegec/auth-tsinghua
- https://github.com/Berrysoft/ClassLibrary
