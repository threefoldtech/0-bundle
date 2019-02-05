# Zero Bundle
runs a mini env flist on linux.

## Operation
zbundle, uses 0-fs to mount an flist then automatically chroot, and run the start point of the env
- once the flist is mounted, zbundle reads the `/etc/env` file
- next, zbundle runs `/etc/start` inside the chroot with the env variables loaded from `/etc/env`
- zbundle shows all the output of the start process on stdout, and stderr and exits once the start process exits

## `env` file format
since NOT all flists will have a shell (`bash`, `sh`, or others) it will not be possible to initialize the
env vars of the `start` process. A special (optional) env file is used to setup the env vars of the start process.

> Note: all env vars from the `env` file can be overridedn from the `zbundle` command line with the `-e` flag

example env file
```bash
# set up the path
PATH=/bin:/usr/bin

# set up the version
VERSION=1.0
```

## `start` file
After loading the `env` from the `/etc/env` file, next `zbundle` will execute the `/etc/start` script.
All the env variables from the `env` file, plus the ones defined from the command line are gonna be available
to the `start` excutable.

- `/etc/start` MUST have `excute` permission (`chmod a+x /etc/start`)
- if `start` is a shell script, the excuting `shebang` must actually be part of the flist otherwise it will not work, because the flist
runs in a chroot, there is noway to access the binaries from the host.

>Note: you can always override the entry point by using `--entry-point` flag

>Note: all command line arguments provided after the flist, are passed to the entry point as arguments

### Notes
- the env file is NOT a shell script
- empty lines and lines starting with `#` are ignored

## Example
```
zbundle --id test https://hub.gig.tech/azmy/bundle-test.flist
```

## Error reporting upstream
If the sandboxed exited with an error, the last 32KB of both stdout, and stderr and collected
along with the exit error, and then reported to ALL provided `report` flag

`report` is a url to the required report endpoint. We only currently support `redis` and `redis+tls`

- for redis, the url formated like `redis://host:port`
- for redis+tls, the url is formated like `redis+tls://host:port`

## Help

```
~ zbundle --help
NAME:
   zbundle - run mini environments from flist

USAGE:
   zbundle [options] <flist> <args>

VERSION:
   1.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --id value, -i value       [required] ID of the sandbox
   --entry-point value        sandbox entry point (default: "/etc/start")
   --env value, -e value      custom environemt variables
   --report value, -r value   report error back on failures to the given url
   --storage value, -s value  storage url to use (default: "ardb://hub.gig.tech:16379")
   --debug, -d                run in debug mode
   --no-exit                  do not terminate (unmount the sandbox) after /etc/start exits
   --help, -h                 show help
   --version, -v              print the version
```

# Owner
[muhamadazmy](https://github.com/muhamadazmy)
