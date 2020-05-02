package ghclient

import (
	"encoding/json"
	"testing"

	"github.com/infrawatch/ci-server-go/pkg/assert"
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
	t.Run("push event handle", func(t *testing.T) {
		webhook := getWebhook()
		testRepoJSON, err := json.Marshal(webhook)
		assert.Ok(t, err)
		gh := NewClient(nil, nil)

		pushEvent := &Push{}

		err = pushEvent.handle(&gh, testRepoJSON)
		if err != nil {
			assert.Ok(t, err)
		}

		var repoName string
		switch n := webhook["repository"].(map[string]interface{})["name"].(type) {
		case string:
			repoName = n
		}
		refName := webhook["ref"].(string)
		ref := gh.repositories[repoName].refs[`"`+refName+`"`]

		assert.Assert(t, (gh.repositories[repoName] != nil), "failed to create repository")
		assert.Assert(t, (ref != nil), "failed to create reference")
		assert.Assert(t, (ref.head != nil), "failed to create commit")
		return
	})
}
