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
Configurations are passed in with environmental variables

*var* | *default* | *description*
---------- |---------- | ----------
GITHUB_USER | - | name of github user that server should run as
OAUTH | - | oauth token for the above mentioned user
ADDRESS | localhost:3000 | address on which to listen for webhooks
NUM_WORKERS | 4 | max number of jobs that can execute in parallel

For results to be posted to github, the configured user must have access to repositories the server is intended to run on.

# Run
```bash
./server
```

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
