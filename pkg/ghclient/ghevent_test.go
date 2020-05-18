package ghclient

import (
	"encoding/json"
	"testing"

	"github.com/pleimer/ci-server-go/pkg/assert"
)

func getWebhook() map[string]interface{} {

	return map[string]interface{}{
		"ref": "refs/head/master",
		"commits": []map[string]interface{}{
			{
				"id":      "test-id",
				"url":     "www.example.com",
				"message": "some-message",
				"author": map[string]interface{}{
					"name":     "my-name",
					"email":    "email@example.com",
					"username": "Codertocat",
				},
			},
		},
		"repository": map[string]interface{}{
			"name": "example-repo",
			"owner": map[string]interface{}{
				"login": "Codertocat",
			},
			"fork": false,
		},
	}
}

func TestHandle(t *testing.T) {
	t.Run("null event", func(t *testing.T) {
		gh := NewClient(nil)

		pushEvent := &Push{}

		err := pushEvent.Handle(&gh, nil)
		assert.Assert(t, (err != nil), "should be error")
	})

	t.Run("two of same event", func(t *testing.T) {
		webhook := getWebhook()
		testRepoJSON, err := json.Marshal(webhook)

		assert.Ok(t, err)
		gh := NewClient(nil)

		pushEvent := &Push{}

		err = pushEvent.Handle(&gh, testRepoJSON)
		assert.Ok(t, err)

		repoName := webhook["repository"].(map[string]interface{})["name"].(string)
		refName := webhook["ref"].(string)
		ref := gh.repositories[repoName].refs[`"`+refName+`"`]

		assert.Assert(t, (gh.repositories[repoName] != nil), "failed to create repository")
		assert.Assert(t, (ref != nil), "failed to create reference")
		assert.Assert(t, (ref.head != nil), "failed to create commit")

		err = pushEvent.Handle(&gh, testRepoJSON)
		if err != nil {
			assert.Ok(t, err)
		}

		assert.Assert(t, (gh.repositories[repoName] != nil), "failed to create repository")
		assert.Assert(t, (ref != nil), "failed to create reference")
		assert.Assert(t, (ref.head != nil), "failed to create commit")
	})

	t.Run("multiple refs", func(t *testing.T) {
		webhook := getWebhook()
		testRepoJSON, err := json.Marshal(webhook)

		assert.Ok(t, err)
		gh := NewClient(nil)

		pushEvent := &Push{}

		err = pushEvent.Handle(&gh, testRepoJSON)
		assert.Ok(t, err)

		repoName := webhook["repository"].(map[string]interface{})["name"].(string)
		refName := webhook["ref"].(string)
		ref := gh.repositories[repoName].refs[`"`+refName+`"`]

		assert.Assert(t, (gh.repositories[repoName] != nil), "failed to create repository")
		assert.Assert(t, (ref != nil), "failed to create reference")
		assert.Assert(t, (ref.head != nil), "failed to create commit")

		webhook["ref"] = "refs/head/branch2"
		testRepoJSON, err = json.Marshal(webhook)

		err = pushEvent.Handle(&gh, testRepoJSON)
		if err != nil {
			assert.Ok(t, err)
		}

		repoName = webhook["repository"].(map[string]interface{})["name"].(string)
		refName = webhook["ref"].(string)
		ref = gh.repositories[repoName].refs[`"`+refName+`"`]

		assert.Assert(t, (gh.repositories[repoName] != nil), "failed to create repository")
		assert.Assert(t, (ref != nil), "failed to create reference")
		assert.Assert(t, (ref.head != nil), "failed to create commit")
	})

	t.Run("new branch", func(t *testing.T) {
		webhook := getWebhook()

		// in new branch, webhook contains no commits in the "commit" mapslice
		// but does contain the commit in the "head_commit" key
		headCommit := webhook["commits"].([]map[string]interface{})[0]
		webhook["commits"] = make([]map[string]interface{}, 0)
		webhook["head_commit"] = headCommit
		testRepoJSON, err := json.Marshal(webhook)
		assert.Ok(t, err)

		gh := NewClient(nil)

		pushEvent := &Push{}

		err = pushEvent.Handle(&gh, testRepoJSON)
		if err != nil {
			assert.Ok(t, err)
		}

		repoName := webhook["repository"].(map[string]interface{})["name"].(string)
		refName := webhook["ref"].(string)
		ref := gh.repositories[repoName].refs[`"`+refName+`"`]

		assert.Assert(t, (gh.repositories[repoName] != nil), "failed to create repository")
		assert.Assert(t, (ref != nil), "failed to create reference")
		assert.Assert(t, (ref.head != nil), "failed to create commit")

	})

	t.Run("corrupt push", func(t *testing.T) {
		webhook := getWebhook()

		webhook["commits"] = make([]map[string]interface{}, 0)
		testRepoJSON, err := json.Marshal(webhook)
		assert.Ok(t, err)

		gh := NewClient(nil)

		pushEvent := &Push{}

		err = pushEvent.Handle(&gh, testRepoJSON)
		if err != nil {
			assert.Assert(t, (err != nil), "should have been an error")
		}
	})

	t.Run("branch merge", func(t *testing.T) {
		webhook := getWebhook()
		testRepoJSON, err := json.Marshal(webhook)

		assert.Ok(t, err)
		gh := NewClient(nil)

		pushEvent := &Push{}

		err = pushEvent.Handle(&gh, testRepoJSON)
		if err != nil {
			assert.Ok(t, err)
		}

		repoName := webhook["repository"].(map[string]interface{})["name"].(string)
		refName := webhook["ref"].(string)
		ref := gh.repositories[repoName].refs[`"`+refName+`"`]

		assert.Assert(t, (gh.repositories[repoName] != nil), "failed to create repository")
		assert.Assert(t, (ref != nil), "failed to create reference")
		assert.Assert(t, (ref.head != nil), "failed to create commit")
	})
}
