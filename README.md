# GoAuthing

[![Build Status](https://travis-ci.org/z4yx/GoAuthing.svg?branch=master)](https://travis-ci.org/z4yx/GoAuthing)
![GPLv3](https://img.shields.io/badge/license-GPLv3-blue.svg)

A commandline Tunet (auth4/6.tsinghua.edu.cn) authentication tool.

## Download Binary

Prebuilt binaries available at https://github.com/z4yx/GoAuthing/releases

## Usage

Simply try `./auth-thu`, then enter your user name and password.

```
NAME:
   auth-thu - Authenticating utility for auth.tsinghua.edu.cn (srun4000)

USAGE:
   auth-thu [-u <username>] [-p <password>] [options]

VERSION:
   0.3

AUTHOR:
   Yuxiang Zhang <yuxiang.zhang@tuna.tsinghua.edu.cn>

GLOBAL OPTIONS:
   --username name, -u name          your TUNET account name
   --password password, -p password  your TUNET password
   --ip value                        authenticating for specified IP address
   --no-check, -n                    skip online checking, always send login request
   --logout, -o                      log out of the online account
   --ipv6, -6                        authenticating for IPv6 (auth6.tsinghua)
   --host value                      use customized hostname of srun4000
   --help, -h                        print the help
   --debug                           print debug messages
   --version, -v                     print the version
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


