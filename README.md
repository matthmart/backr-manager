# Backr Manager

[![Build Status](https://travis-ci.org/matthmart/backr-manager.svg?branch=master)](https://travis-ci.org/matthmart/backr-manager)
[![Go Report Card](https://goreportcard.com/badge/github.com/matthmart/backr-manager)](https://goreportcard.com/report/github.com/matthmart/backr-manager)

Backr Manager is a tool created to manage backup files stored into an object storage solution. 

**Table of content**

- [Presentation](#presentation)
  - [Purpose](#purpose)
  - [Architecture](#architecture)
- [How to use](#how-to-use)
  - [Start the server](#start-the-server)
  - [Configuration](#configuration)
- [Building](#building)
- [Readmap](#roadmap)
- [License](#license)

## Presentation

### Purpose

This tool manages the lifecycle of backup files, using some fine-grained control rules. You may want to keep some recent files of the last 3 days and some older files (e.g. 15 days at least), to be able to turn back time.

One of the key features of the tool is to detect some issues with the backup routine. Backr Manager is able to monitor and send alerts when expected files are missing or when a file size is a lot smaller than the previous file.

### Architecture

The main entities are *files* and *projects*. Projects are configured by the user, allowing to define lifecycle rules for files stored into a same folder. So a folder is linked to a project. Files are stored in an object storage service like S3. Each similar file is expected to be stored in a specific folder. File uploading is not the responsability of Backr Manager.

So the requirements/assumptions are:

- similar files must be stored in a folder
- a project is linked to a folder and lifecycle rules will be applied to the files of this folder

When the daemon is starting, the process manager runs periodically and checks for file changes in each configured project. It detects potential issues (small size, missing file, etc), send alerts if needed and remove files not needed anymore (if the rules are fulfilled).

A gRPC API is exposed to communicate with the process manager. It allows to manage projects, list files, get a temporary URL to download a file. The user must be authenticated to interact with the API. User management is also integrated to the API.

A CLI client is available to interact with the API.

## How to use

### Start the server

```
backr-manager daemon start
```

### Configuration

The daemon uses a config file written in TOML. 

```
[s3]
bucket =  "bucket-name"
endpoint =  "play.min.io"
access_key =  ""
secret_key =  ""
use_tls =  true

[bolt]
filepath = "bolt.db"

# to generate auth token
[auth]
jwt_secret = "a_very_secure_key"
```

You can specify a path for the config file using

```
backr-manager daemon start --config PATH
```

The config can also be set by environment variables (documentation in progress).

### How to use the CLI client

To get available commands, just type `backrctl -h` in your terminal.

```
$ backrctl -h
CLI tool to interact with a running Backr manager instance

Usage:
  backrctl [command]

Available Commands:
  account     Manage user accounts
  file        Manage files
  help        Help about any command
  login       Login using username and password, and save token into a file in $HOME directory (.backr_auth)
  project     Manage projects

Flags:
      --endpoint string   Endpoint of the Backr instance (default "127.0.0.1:3000")
  -h, --help                    help for backrctl

Use "backrctl [command] --help" for more information about a command.
```

## First launch

The daemon must be running and the API must be accessible. By default, the client will try to connect to `127.0.0.1:3000`. You can change this endpoint using the flag `--endpoint` or the environment variable `BACKRCTL_ENDPOINT`.

To get started, just follow these steps:

```
$ backrctl account create --username john
password:
yR=fl?nFgh+q7?Ll
```

An account is created for `john`, a password is automatically generated and it must be kept securely. It is not stored and you will not be able to get it again.
Because this is the first launch, authentication is not required until an account is created. So just after this step, authentication is enabled and your account is the one than can authenticate against the API. **If you lose your password, you will need to remove backr-manager DB.**

Next, create a project:

```
$ backrctl project create --name project1 --rule 3.1 --rule 2.15
```

This command will create a project named `project1` and configure 2 lifecycles rules (the unit is day):

 - `3.1`: we want at least 3 files with a minimum age of 1 day. So the selected files will expire after 3 days (1 * 3 days)
 - `2.15`: we want at least 2 files with a minimum age of 15 days. So the selected files will expire after 30 days (2 * 15 days)
 
If there is no error, the expired files will be removed.

You can check the project is correctly created using:

```
$ backrctl project ls
NAME       CREATED_AT                       RULES (count.min_age)  
project1   2019-08-04 00:09:59 +0200 CEST   3.1 2.15
```

To get more info on the project, use this command:

```
$ backrctl project get project1 -a
PROJECT NAME   CREATED AT                       
project1       2019-08-04 00:09:59 +0200 CEST   

3.1 (next: 2019-08-16 02:59:47 +0200 CEST)
PATH                     DATE                             EXPIRE AT                        SIZE   ERROR                              
project1/file12.tar.gz   2019-08-15 00:59:47 +0200 CEST   2019-08-16 00:59:47 +0200 CEST   11     -

2.15 (next: 2019-08-16 02:59:47 +0200 CEST)
PATH                     DATE                             EXPIRE AT                        SIZE   ERROR                              
project1/file12.tar.gz   2019-08-15 00:59:47 +0200 CEST   2019-08-16 00:59:47 +0200 CEST   11     -
```

When you need to download a file, you can use this command:

```
$ backrctl file url project1/file12.tar.gz
<a very long URL>
```

# Building

To build binaries, you can use the Makefile.

```
$ make build
```

To cross-compile the binaries:

```
$ make build_all
```

# Roadmap

The tool is still missing some important features:

- a true notifier/alert system. The first one will use Slack as channel.
- add more unit tests

# License

MIT