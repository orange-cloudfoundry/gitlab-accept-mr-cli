# gitlab-accept-mr

Accept automatically all opened merge request in a project on Gitlab.

## Installation

### On *nix system

You can install this via the command-line with either `curl` or `wget`.

#### via curl

```bash
$ sh -c "$(curl -fsSL https://raw.github.com/ArthurHlt/gitlab-accept-mr-cli/master/bin/install.sh)"
```

#### via wget

```bash
$ sh -c "$(wget https://raw.github.com/ArthurHlt/gitlab-accept-mr-cli/master/bin/install.sh -O -)"
```

### On windows

You can install it by downloading the `.exe` corresponding to your cpu from releases page: https://github.com/ArthurHlt/gitlab-accept-mr-cli/releases .
Alternatively, if you have terminal interpreting shell you can also use command line script above, it will download file in your current working dir.

### From go command line

Simply run in terminal:

```bash
$ go get github.com/ArthurHlt/gitlab-accept-mr-cli
```

## Usage

```
NAME:
   accept-mr - Automatically accept Merge Request on project

USAGE:
   accept-mr [global options] command [command options] [arguments...]

VERSION:
   1.0.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --url value, -u value              Url to your gitlab [$GITLAB_URL]
   --token value, -t value            User token to access the api [$GITLAB_TOKEN]
   --project value, -p value          Project name where accepting mr (e.g.: owner/repo) [$GITLAB_PROJECT]
   --pipeline-name value, --pn value  Set a default pipeline name when using on-build-succeed option (default: "accept-mr")
   --failed-on-error, -e              If true accept in error exit with status code > 0
   --insecure, -k                     Ignore certificate validation
   --log-json, -j                     Write log in json
   --no-color                         Logger will not display colors
   --on-build-succeed, --bs           Merge request will automatically accepted if pipeline succeeded
   --help, -h                         show help
   --version, -v                      print the version
```