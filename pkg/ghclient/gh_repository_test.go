package ghclient

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/infrawatch/ci-server-go/pkg/assert"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Int63()%int64(len(letters))]
	}
	return string(b)
}

func genCommits(substr string, start int, num int) (*Commit, *Commit) {
	if substr == "" {
		substr = randString(10)
	}

	prevCommit := &Commit{
		Sha: substr + "-" + strconv.Itoa(start),
	}

	tail := prevCommit
	var newCommit *Commit
	for i := 0; i < num-1; i++ {
		newCommit = &Commit{
			Sha: substr + "-" + strconv.Itoa(i+start+1),
		}
		prevCommit.linkChild(newCommit)
		prevCommit = newCommit
	}
	return prevCommit, tail // head, tail
}

// logCommits debug func
func logCommits(t *testing.T, head *Commit) {
	for c := head; c != nil; c = c.parent {
		t.Log(c.String())
	}
}

func compareCommits(t *testing.T, cExp *Commit, cRec *Commit) {

	for {
		assert.Equals(t, cExp.Sha, cRec.Sha)

		cExp = cExp.parent
		cRec = cRec.parent

		if cExp == nil || cRec == nil {
			break
		}
	}
}

func TestRegisterCommits(t *testing.T) {

	repoUT := Repository{
		refs: make(map[string]*Reference),
	}
	t.Run("register to empty cache", func(t *testing.T) {
		head, _ := genCommits("original", 0, 3)

		repoUT.registerCommits(head, "refs/head/master")
		cGen := head
		cUT := repoUT.refs["refs/head/master"].head

		compareCommits(t, cGen, cUT)
	})

	t.Run("crossover", func(t *testing.T) {
		gen, _ := genCommits("original", 1, 1)
		gen2, genTail2 := genCommits("crossover", 0, 3)
		gen.linkChild(genTail2)

		cmp, _ := genCommits("original", 0, 2)
		cmp2, cmpTail2 := genCommits("crossover", 0, 3)
		cmpTail2.parent = cmp.parent.parent
		cmp.linkChild(cmpTail2)

		repoUT.registerCommits(gen2, "refs/head/master")

		cUT := repoUT.refs["refs/head/master"].head
		compareCommits(t, cmp2, cUT)
	})

	t.Run("no crossover", func(t *testing.T) {
		gen, _ := genCommits("nocrossover", 0, 3)
		repoUT.registerCommits(gen, "refs/head/master")

		cmp, _ := genCommits("nocrossover", 0, 3)
		cUT := repoUT.refs["refs/head/master"].head

		compareCommits(t, cmp, cUT)
	})
}
