# GoAuthing

Authenticating utility for auth4/6.tsinghua.edu.cn

## Usage

Simply try `./auth-thu`, then enter your user name and password.

```
NAME:
   auth-thu - Authenticating utility for auth.tsinghua.edu.cn

USAGE:
   auth-thu [-u <username>] [-p <password>] [options]

VERSION:
   0.1

AUTHOR:
   Yuxiang Zhang <yuxiang.zhang@tuna.tsinghua.edu.cn>

GLOBAL OPTIONS:
   --username name, -u name          your TUNET account name
   --password password, -p password  your TUNET password
   --no-check, -n                    skip online checking, always send login request
   --logout, -o                      log out of the online account
   --ipv6, -6                        authenticating for IPv6 (auth6)
   --help, -h                        print the help
   --debug                           print debug messages
   --version, -v                     print the version
```

## Build

```
go build -o auth-thu github.com/z4yx/GoAuthing/cli
```
