package ghclient

import (
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Int63()%int64(len(letters))]
	}
	return string(b)
}

func genCommits(substr string, num int) []Commit {
	if substr == "" {
		substr = randString(10)
	}
	var sCommit []Commit = nil
	for i := 0; i < num; i++ {
		sCommit = append(sCommit, Commit{
			sha: substr + "-" + strconv.Itoa(i),
		})
	}
	return sCommit
}

func commitsString(commits []Commit) string {
	var sb strings.Builder
	for _, commit := range commits {
		sb.WriteString(commit.sha)
		sb.WriteString(" ")
	}
	return sb.String()
}

func TestCacheRegister(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	//put some commits in que.
	// Gen new commits with some crossover of olds
	// store and make sure all are there with no duplicates

	t.Run("1 duplicate incoming commit", func(t *testing.T) {
		cache := CommitCache{}
		origCommits := genCommits("original", 3)
		incCommits := genCommits("incoming", 3)

		cache.Register(origCommits)
		incCommits[0] = origCommits[len(origCommits)-1]
		correctRes := append(origCommits[0:2], incCommits...)

		cache.Register(incCommits)

		if !reflect.DeepEqual(cache.commits, correctRes) {
			t.Error("Failed: structs not equal")
			t.Log("Expected: " + commitsString(correctRes))
			t.Log("Received: " + commitsString(cache.commits))
		}
	})
}
