package ghclient

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/pleimer/ci-server-go/pkg/assert"
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
		prevCommit.setChild(newCommit)
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

	refUT := &Reference{}
	t.Run("blank slate", func(t *testing.T) {
		gen, _ := genCommits("original", 0, 3)

		refUT.Register(gen)
		cmp, _ := genCommits("original", 0, 3)

		compareCommits(t, cmp, refUT.head)
	})

	t.Run("crossover", func(t *testing.T) {
		gen, _ := genCommits("original", 1, 1)
		gen2, genTail2 := genCommits("crossover", 0, 3)
		gen.setChild(genTail2)

		cmp, _ := genCommits("original", 0, 2)
		cmp2, cmpTail2 := genCommits("crossover", 0, 3)
		cmpTail2.parent = cmp.parent.parent
		cmp.setChild(cmpTail2)

		refUT.Register(gen2)

		compareCommits(t, cmp2, refUT.head)
	})

	t.Run("no crossover", func(t *testing.T) {
		gen, _ := genCommits("nocrossover", 0, 3)
		refUT.Register(gen)

		cmp, _ := genCommits("nocrossover", 0, 3)

		compareCommits(t, cmp, refUT.head)
	})
}
