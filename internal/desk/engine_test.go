package desk

import (
	"reflect"
	"testing"
)

func mustNew(t *testing.T, ids ...int) *Desk {
	t.Helper()
	d, err := New(ids)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return d
}

func submitAll(t *testing.T, d *Desk, r *Round, o Outcome) {
	t.Helper()
	full := []ResultInput{}
	for _, g := range r.Games {
		if g.Outcome == OutcomeBye {
			continue
		}
		full = append(full, ResultInput{A: g.A, B: g.B, Outcome: o})
	}
	if _, err := d.SubmitResults(full); err != nil {
		t.Fatalf("submit all: %v", err)
	}
}

func pairsOf(r *Round) [][2]int {
	var out [][2]int
	for _, g := range r.Games {
		if g.Outcome == OutcomeBye {
			continue
		}
		out = append(out, [2]int{g.A, g.B})
	}
	return out
}

func TestNewValidatesPlayers(t *testing.T) {
	if _, err := New([]int{1, 1}); err == nil {
		t.Fatal("expected duplicate id error")
	}
	if _, err := New([]int{0}); err == nil {
		t.Fatal("expected non-positive id error")
	}
	ids := make([]int, 17)
	for i := range ids {
		ids[i] = i + 1
	}
	if _, err := New(ids); err == nil {
		t.Fatal("expected too-many-players error")
	}
}

// TestNonGreedyMatching sets scores so the objective favors pairing
// adjacent scores, and a rematch constraint forces a globally re-planned
// solution rather than a per-player greedy one.
func TestNonGreedyMatching(t *testing.T) {
	d := mustNew(t, 1, 2, 3, 4)
	// Craft scores 0,2,3,4 via a fake confirmed round: give wins manually.
	d.Rounds = []*Round{{
		Number: 1,
		Games: []Game{
			{A: 2, B: 1, Outcome: OutcomeAWin}, // 2: 2 pts
			{A: 3, B: 4, Outcome: OutcomeDraw}, // 3,4: 1 pt each... adjust below
		},
		Confirmed: true,
	}}
	// scores: 1=0, 2=2, 3=1, 4=1
	// No rematch possible for 2 (played 1) and 3-4 played each other.
	// Feasible matchings avoiding {1-2, 3-4}:
	//   (1,3)(2,4): |0-1|+|2-1| = 2
	//   (1,4)(2,3): |0-1|+|2-1| = 2
	// Tie -> lexicographically smallest: (1,3)(2,4).
	r, err := d.GenerateRound()
	if err != nil {
		t.Fatalf("GenerateRound: %v", err)
	}
	want := [][2]int{{1, 3}, {2, 4}}
	if got := pairsOf(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("pairs = %v, want %v", got, want)
	}
	if r.Objective != 2 {
		t.Fatalf("objective = %d, want 2", r.Objective)
	}
}

func TestMatchingMinimizesTotalDiff(t *testing.T) {
	d := mustNew(t, 1, 2, 3, 4)
	// scores: 1=0, 2=2, 3=4, 4=6 via fabricated confirmed rounds.
	d.Rounds = []*Round{{
		Number: 1,
		Games: []Game{
			{A: 2, B: 1, Outcome: OutcomeAWin},
			{A: 4, B: 3, Outcome: OutcomeAWin},
		},
		Confirmed: true,
	}, {
		Number: 2,
		Games: []Game{
			{A: 3, B: 1, Outcome: OutcomeAWin}, // 3: 2
			{A: 4, B: 2, Outcome: OutcomeAWin}, // 4: 4
		},
		Confirmed: true,
	}, {
		Number: 3,
		Games: []Game{
			{A: 3, B: 2, Outcome: OutcomeAWin}, // 3: 4
			{A: 4, B: 1, Outcome: OutcomeAWin}, // 4: 6
		},
		Confirmed: true,
	}}
	// All of 1-2,3-4,1-3,2-4,2-3,1-4 played except... played: 1-2,3-4,1-3,2-4,2-3,1-4 = all!
	// Remove the constraint by using fresh players instead.
	d = mustNew(t, 1, 2, 3, 4)
	d.Players[1].HadBye = false
	// Inject scores directly through rounds is messy; use bestMatching unit test.
	scores := map[int]int{1: 0, 2: 2, 3: 4, 4: 6}
	past := map[int]map[int]bool{1: {}, 2: {}, 3: {}, 4: {}}
	m, obj, err := bestMatching([]int{1, 2, 3, 4}, scores, past)
	if err != nil {
		t.Fatalf("bestMatching: %v", err)
	}
	// (1,2)(3,4): 2+2=4 is the unique minimum.
	if obj != 4 || !reflect.DeepEqual([][2]int(m), [][2]int{{1, 2}, {3, 4}}) {
		t.Fatalf("got %v obj %d", m, obj)
	}
}

func TestNoSolutionKeepsByeAndFails(t *testing.T) {
	d := mustNew(t, 1, 2, 3)
	// 1-2 and 1-3 already played; with 3 players one gets a bye and the
	// other two must play, but 2-3 also played -> no solution.
	d.Rounds = []*Round{{
		Number: 1,
		Games: []Game{
			{A: 1, B: 2, Outcome: OutcomeAWin},
			{A: 3, Outcome: OutcomeBye},
		},
		Bye:       3,
		Confirmed: true,
	}, {
		Number: 2,
		Games: []Game{
			{A: 1, B: 3, Outcome: OutcomeAWin},
			{A: 2, Outcome: OutcomeBye},
		},
		Bye:       2,
		Confirmed: true,
	}, {
		Number: 3,
		Games: []Game{
			{A: 2, B: 3, Outcome: OutcomeDraw},
			{A: 1, Outcome: OutcomeBye},
		},
		Bye:       1,
		Confirmed: true,
	}}
	d.Players[1].HadBye = true
	d.Players[2].HadBye = true
	d.Players[3].HadBye = true
	before := len(d.Rounds)
	if _, err := d.GenerateRound(); err == nil {
		t.Fatal("expected no-solution error")
	}
	if len(d.Rounds) != before {
		t.Fatal("failed pairing must not append a round")
	}
}

func TestByeSelectionAndRotation(t *testing.T) {
	d := mustNew(t, 1, 2, 3)
	// All scores 0: bye goes to lowest id without a bye -> 1.
	r1, err := d.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	if r1.Bye != 1 {
		t.Fatalf("bye = %d, want 1", r1.Bye)
	}
	if _, err := d.SubmitResults([]ResultInput{{A: 2, B: 3, Outcome: OutcomeAWin}}); err != nil {
		t.Fatal(err)
	}
	// scores: 1=2 (bye), 2=2, 3=0. Never-bye: {2,3}; lowest score -> 3.
	r2, err := d.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	if r2.Bye != 3 {
		t.Fatalf("bye = %d, want 3", r2.Bye)
	}
}

func TestPendingRoundBlocksNextPairing(t *testing.T) {
	d := mustNew(t, 1, 2)
	if _, err := d.GenerateRound(); err != nil {
		t.Fatal(err)
	}
	if _, err := d.GenerateRound(); err == nil {
		t.Fatal("expected error: current round not submitted")
	}
}

func TestRejectedResultsLeaveStateUntouched(t *testing.T) {
	d := mustNew(t, 1, 2, 3, 4)
	r, err := d.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	// Invalid outcome.
	bad := []ResultInput{{A: r.Games[0].A, B: r.Games[0].B, Outcome: "win"}}
	if _, err := d.SubmitResults(bad); err == nil {
		t.Fatal("expected invalid outcome error")
	}
	// Missing game.
	partial := []ResultInput{{A: r.Games[0].A, B: r.Games[0].B, Outcome: OutcomeAWin}}
	if _, err := d.SubmitResults(partial); err == nil {
		t.Fatal("expected missing-results error")
	}
	// Unknown game.
	wrong := append([]ResultInput{}, partial...)
	wrong = append(wrong, ResultInput{A: 1, B: 3, Outcome: OutcomeDraw})
	if pairsContain(pairsOf(r), [2]int{1, 3}) {
		wrong[1] = ResultInput{A: 1, B: 4, Outcome: OutcomeDraw}
	}
	if _, err := d.SubmitResults(wrong); err == nil {
		t.Fatal("expected unknown-game error")
	}
	// State untouched: round still pending, all scores zero.
	if d.pendingRound() == nil {
		t.Fatal("round should still be pending")
	}
	for id, s := range d.scores() {
		if s != 0 {
			t.Fatalf("player %d score = %d after rejected batch", id, s)
		}
	}
	// Now a valid full batch applies.
	full := []ResultInput{}
	for _, g := range r.Games {
		full = append(full, ResultInput{A: g.A, B: g.B, Outcome: OutcomeAWin})
	}
	if _, err := d.SubmitResults(full); err != nil {
		t.Fatalf("valid batch rejected: %v", err)
	}
	// Confirmed round cannot be resubmitted.
	if _, err := d.SubmitResults(full); err == nil {
		t.Fatal("expected error on resubmitting confirmed round")
	}
}

func pairsContain(pairs [][2]int, p [2]int) bool {
	for _, q := range pairs {
		if q == p {
			return true
		}
	}
	return false
}

func TestWithdrawKeepsHistoryAndSkipsPairing(t *testing.T) {
	d := mustNew(t, 1, 2, 3, 4, 5)
	r, err := d.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	// Cannot withdraw mid-round.
	if err := d.Withdraw(1); err == nil {
		t.Fatal("expected withdraw-during-round error")
	}
	submitAll(t, d, r, OutcomeAWin)
	if err := d.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	r2, err := d.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pairsOf(r2) {
		if p[0] == 1 || p[1] == 1 {
			t.Fatal("withdrawn player was paired")
		}
	}
	// History preserved: player 1 still ranked with earned score.
	found := false
	for _, s := range d.Standings() {
		if s.ID == 1 {
			found = true
			if s.Score != 2 || !s.Withdrawn {
				t.Fatalf("standing = %+v", s)
			}
		}
	}
	if !found {
		t.Fatal("withdrawn player missing from standings")
	}
}

func TestStandingsTieBreak(t *testing.T) {
	d := mustNew(t, 1, 2, 3, 4)
	// Round 1: 1 beats 2, 3 beats 4. Round 2: 1-3 draw, 2-4 draw.
	d.Rounds = []*Round{{
		Number: 1,
		Games: []Game{
			{A: 1, B: 2, Outcome: OutcomeAWin},
			{A: 3, B: 4, Outcome: OutcomeAWin},
		},
		Confirmed: true,
	}, {
		Number: 2,
		Games: []Game{
			{A: 1, B: 3, Outcome: OutcomeDraw},
			{A: 2, B: 4, Outcome: OutcomeDraw},
		},
		Confirmed: true,
	}}
	// scores: 1=3, 3=3, 2=1, 4=1.
	// Buchholz: 1 -> opp {2,3}: 1+3=4. 3 -> opp {4,1}: 1+3=4. Tie -> id asc.
	st := d.Standings()
	wantOrder := []int{1, 3, 2, 4}
	for i, id := range wantOrder {
		if st[i].ID != id || st[i].Rank != i+1 {
			t.Fatalf("standings[%d] = %+v, want id %d", i, st[i], id)
		}
	}
	if st[0].Buchholz != 4 {
		t.Fatalf("buchholz = %d, want 4", st[0].Buchholz)
	}
}
