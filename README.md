#  ci-server-go
This is a small ci runner that reacts to github webhooks and runs different types of jobs depending on the webhook type. The primary job involves running commands in a `ci.yml` file in the top level directory of a repository. For example, on a push webhook, the runner will download the repository, run the `ci.yml` specification while continually updating a github gist with the output and finally post a pass or fail status to the commit that triggered the job. 

Another type of job might run on a comment webhook. If the comment on a PR contains a specific keyword, the job can re-run the sequence above. 


# Build
Requires go v1.11 or higher
```
git clone https://github.com/pleimer/ci-server-go.git
cd ci-server-go
go get ./...
go build cmd/server.go
```

# Configuration
Configurations are loaded by default from `/etc/ci-server-go.conf.yaml`. Custom locations can be specified with the `-config` option.

A user with admin access to the repositories intended to use this CI must be configured. The server runs under this user.

```yaml
github:
    user:   # username with admin privledges to monitored repositories
    oauth:  # oauth token for above user

listener: 
    address: # [Optional] listen for webhooks here. Default: localhost:3000

logger:
    level:  # [Optional] DEBUG,INFO,WARN,ERROR,NONE. Default: INFO
    target: # [Optional] 'console' or filepath. Default: 'console'

runner:
    numWorkers: # [Optional] number of jobs that can run in parallel. Default: 4
    authorizedUsers: # users athorized to run CI jobs
        - user1
        - user2
```

# Run
```bash
./server
```

## Options
Option | Description
-|-
-help | show help menu
-config | specify config file location

# ci.yml

## magic variables
Magic variables contain information about the job environment that commands in `ci.yml` can access. For example, a ci script may want some information about the commit that triggered its run. In this case, the sha of that commit can be accessed with the `__commit__ ` magic variable. Magic variables must be stored to an environmental variable to be accessed by the main script sections in `ci.yml`. Therefor, to print the sha of the commit, a `ci.yml` might look like the following:

```yaml

gloabl:
    timeout: 300
    env:
        COMMIT_SHA: __commit__
script:
    - echo "$COMMIT_SHA"
```
