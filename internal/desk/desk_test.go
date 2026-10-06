package desk

import (
	"reflect"
	"testing"
)

func mustNew(t *testing.T, players ...int) *Tournament {
	t.Helper()
	tr, err := New(players)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return tr
}

func TestFirstRoundLexicographicPairs(t *testing.T) {
	tr := mustNew(t, 4, 3, 2, 1)
	r, err := tr.GenerateRound()
	if err != nil {
		t.Fatalf("GenerateRound: %v", err)
	}
	want := []Pair{{1, 2}, {3, 4}}
	if !reflect.DeepEqual(r.Pairs, want) {
		t.Fatalf("pairs = %v, want %v", r.Pairs, want)
	}
	if r.Objective != 0 || r.Bye != 0 {
		t.Fatalf("objective=%d bye=%d, want 0/0", r.Objective, r.Bye)
	}
}

// TestNonGreedyMatching blocks the greedy first choice so that only a
// globally searched matching succeeds: played (1,2) and (2,4).
// Greedy would take (1,3) and get stuck with (2,4); the valid answer is
// (1,4),(2,3).
func TestNonGreedyMatching(t *testing.T) {
	tr := mustNew(t, 1, 2, 3, 4)
	tr.Opponents[1] = []int{2}
	tr.Opponents[2] = []int{1, 4}
	tr.Opponents[4] = []int{2}
	r, err := tr.GenerateRound()
	if err != nil {
		t.Fatalf("GenerateRound: %v", err)
	}
	want := []Pair{{1, 4}, {2, 3}}
	if !reflect.DeepEqual(r.Pairs, want) {
		t.Fatalf("pairs = %v, want %v", r.Pairs, want)
	}
}

// TestNonGreedyObjective: greedy by id order ((1,2),(3,4)) costs 1+1=2,
// but the global optimum ((1,4),(2,3)) costs 3+1=4... so here we verify the
// minimizer picks the true minimum even when it is not the lexicographic
// first: scores 1:0, 2:1, 3:2, 4:3 give options 2, 4, 4; optimum is
// (1,2),(3,4) with objective 2.
func TestObjectiveMinimized(t *testing.T) {
	tr := mustNew(t, 1, 2, 3, 4)
	tr.Scores = map[int]int{1: 0, 2: 1, 3: 2, 4: 3}
	r, err := tr.GenerateRound()
	if err != nil {
		t.Fatalf("GenerateRound: %v", err)
	}
	if r.Objective != 2 {
		t.Fatalf("objective = %d, want 2", r.Objective)
	}
	want := []Pair{{1, 2}, {3, 4}}
	if !reflect.DeepEqual(r.Pairs, want) {
		t.Fatalf("pairs = %v, want %v", r.Pairs, want)
	}
}

func TestByeSelection(t *testing.T) {
	tr := mustNew(t, 1, 2, 3)
	tr.Scores = map[int]int{1: 4, 2: 2, 3: 2}
	r, err := tr.GenerateRound()
	if err != nil {
		t.Fatalf("GenerateRound: %v", err)
	}
	if r.Bye != 2 { // lowest score, then lowest id
		t.Fatalf("bye = %d, want 2", r.Bye)
	}
	if len(r.Pairs) != 1 || r.Pairs[0] != (Pair{1, 3}) {
		t.Fatalf("pairs = %v, want [{1 3}]", r.Pairs)
	}
}

func TestByeNotRepeated(t *testing.T) {
	tr := mustNew(t, 1, 2, 3)
	tr.Scores = map[int]int{1: 0, 2: 5, 3: 5}
	tr.Byes[1] = true // 1 already had a bye
	r, err := tr.GenerateRound()
	if err != nil {
		t.Fatalf("GenerateRound: %v", err)
	}
	if r.Bye == 1 {
		t.Fatalf("bye repeated for player 1")
	}
}

func TestNoPairingFails(t *testing.T) {
	tr := mustNew(t, 1, 2)
	tr.Opponents[1] = []int{2}
	tr.Opponents[2] = []int{1}
	if _, err := tr.GenerateRound(); err != ErrNoPairing {
		t.Fatalf("err = %v, want ErrNoPairing", err)
	}
	if len(tr.Rounds) != 0 {
		t.Fatalf("failed pairing must not append a round")
	}
}

func TestOddNoPairingKeepsByeUnchanged(t *testing.T) {
	tr := mustNew(t, 1, 2, 3)
	// All pairs already played: no matching exists for the remaining two.
	tr.Opponents[1] = []int{2, 3}
	tr.Opponents[2] = []int{1, 3}
	tr.Opponents[3] = []int{1, 2}
	if _, err := tr.GenerateRound(); err != ErrNoPairing {
		t.Fatalf("err = %v, want ErrNoPairing", err)
	}
	if len(tr.Rounds) != 0 {
		t.Fatalf("round must not be recorded on failure")
	}
}
