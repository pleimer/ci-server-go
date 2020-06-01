#  ci-server-go
This is a small ci runner that listens for webhooks from github repositories, executes `ci.yml` in the top level directory of the repostiroy and posts the results to whatever commit the incoming webhook corresponds to. Sometimes, a webhook may envolve several commits, in which case the most recent one is executed.

Stdout of commands run in `ci.yml` are written to both a logfile on the host machine and a gist file on the user the server runs with. The gist file can be accessed by viewing the commit status. 

# Build
Requires go v1.11 or higher
```
git clone https://github.com/pleimer/ci-server-go.git
cd ci-server-go
go get ./...
go build cmd/server.go
```

# Configuring
Configurations are passed in with environmental variables

*var* | *default* | *description*
---------- |---------- | ----------
GITHUB_USER | - | name of github user that server should run as
OAUTH | - | oauth token for the above mentioned user
ADDRESS | localhost:3000 | address on which to listen for webhooks
NUM_WORKERS | 4 | max number of jobs that can execute in parallel

For results to be posted to github, the configured user must have access to repositories the server is intended to run on.



