# GoAuthing

[![Build Status](https://travis-ci.org/z4yx/GoAuthing.svg?branch=master)](https://travis-ci.org/z4yx/GoAuthing)
![GPLv3](https://img.shields.io/badge/license-GPLv3-blue.svg)

A commandline Tunet (auth4/6.tsinghua.edu.cn, Tsinghua-IPv4) authentication tool.

## Download Binary

Prebuilt binaries available at https://github.com/z4yx/GoAuthing/releases

## Usage

Simply try `./auth-thu`, then enter your user name and password.

```
NAME:
   auth-thu - Authenticating utility for Tsinghua

USAGE:
   auth-thu [options]
   auth-thu [options] auth [auth_options]
   auth-thu [options] login
   auth-thu [options] logout

VERSION:

   1.3

AUTHORS:
  Yuxiang Zhang <yuxiang.zhang@tuna.tsinghua.edu.cn>
  Nogeek <ritou11@gmail.com>

COMMANDS:
     auth    (default) Auth via auth4/6.tsinghua
       OPTIONS:
           --ip value      authenticating for specified IP address
           --no-check, -n  skip online checking, always send login request
           --logout, -o    log out of the online account
           --ipv6, -6      authenticating for IPv6 (auth6.tsinghua)
           --host value    use customized hostname of srun4000
           --insecure      use http instead of https
     login   Login via net.tsinghua
     logout  Logout via net.tsinghua

GLOBAL OPTIONS:
   --username name, -u name          your TUNET account name
   --password password, -p password  your TUNET password
   --config-file path, -c path       path to your config file, default ~/.auth-thu
   --debug                           print debug messages
   --help, -h                        print the help
   --version, -v                     print the version
```

Write a config file to store your username & password or other options in the following format.
The default location of config file is `~/.auth-thu`.

```
{
  "username": your-username,
  "password": your-password,
  "host": "auth4.tsinghua.edu.cn",
  "ip": "166.xxx.xx.xx",
  "debug": false,
  "useV6": false,
  "noCheck": false,
  "insecure": false
}
```

## Build

```
go get -u github.com/z4yx/GoAuthing/cli
go build -o auth-thu github.com/z4yx/GoAuthing/cli
```

## Acknowledgments

This project was inspired by the following projects:

- https://github.com/jiegec/auth-tsinghua
- https://github.com/Berrysoft/ClassLibrary
