# Github exporter

Exports your commit, pull_request, issues, release history to stdout or file

## Usage

```
NAME:
   github-export - Export GitHub user activity

USAGE:
   github-export [global options] command [command options]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --output value, -o value  Output file path (default: "github-export.json")
   --token value, -t value   Github API access token [$GITHUB_TOKEN]
   --format value, -f value  Output format (json, csv, txt)
   --kind value, -k value    Kind of data to export (commits, pull_requests, issues, releases) (default: "commits")
   --mode value, -m value    Use the Github events API
   --help, -h                show help
```
