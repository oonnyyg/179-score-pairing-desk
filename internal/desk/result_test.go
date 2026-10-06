package desk

import (
	"reflect"
	"testing"
)

func TestSubmitRejectsPartialBatch(t *testing.T) {
	tr := mustNew(t, 1, 2, 3, 4)
	if _, err := tr.GenerateRound(); err != nil {
		t.Fatal(err)
	}
	// Missing one game: whole batch must be rejected, no scores applied.
	_, err := tr.SubmitResults([]Result{{Pair: Pair{1, 2}, Winner: 1}})
	if err == nil {
		t.Fatal("expected error for incomplete batch")
	}
	if tr.Scores[1] != 0 || tr.Scores[2] != 0 {
		t.Fatalf("scores changed after rejected batch: %v", tr.Scores)
	}
	if tr.currentRound().Confirmed {
		t.Fatal("round confirmed after rejected batch")
	}
	// Invalid winner also rejected atomically.
	_, err = tr.SubmitResults([]Result{
		{Pair: Pair{1, 2}, Winner: 1},
		{Pair: Pair{3, 4}, Winner: 2},
	})
	if err == nil {
		t.Fatal("expected error for invalid winner")
	}
	if tr.Scores[1] != 0 {
		t.Fatal("partial scoring after rejected batch")
	}
	// Valid full batch applies.
	if _, err := tr.SubmitResults([]Result{
		{Pair: Pair{1, 2}, Winner: 1},
		{Pair: Pair{3, 4}, Winner: 0},
	}); err != nil {
		t.Fatalf("valid submit: %v", err)
	}
	if tr.Scores[1] != 2 || tr.Scores[3] != 1 || tr.Scores[4] != 1 {
		t.Fatalf("scores = %v", tr.Scores)
	}
	// Confirmed round cannot be submitted again.
	if _, err := tr.SubmitResults([]Result{
		{Pair: Pair{1, 2}, Winner: 2},
		{Pair: Pair{3, 4}, Winner: 3},
	}); err != ErrRoundConfirmed {
		t.Fatalf("err = %v, want ErrRoundConfirmed", err)
	}
	if tr.Scores[2] != 0 {
		t.Fatal("double submission changed scores")
	}
}

func TestNoNextRoundBeforeResults(t *testing.T) {
	tr := mustNew(t, 1, 2, 3, 4)
	if _, err := tr.GenerateRound(); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.GenerateRound(); err != ErrRoundOpen {
		t.Fatalf("err = %v, want ErrRoundOpen", err)
	}
}

func TestStandingsTieBreak(t *testing.T) {
	tr := mustNew(t, 1, 2, 3, 4)
	// Round 1: (1,2) draw, (3,4) draw -> everyone 1 point.
	if _, err := tr.GenerateRound(); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.SubmitResults([]Result{
		{Pair: Pair{1, 2}, Winner: 0},
		{Pair: Pair{3, 4}, Winner: 0},
	}); err != nil {
		t.Fatal(err)
	}
	// Round 2: objective-minimal pairing of equal scores is (1,3),(2,4)
	// or (1,4),(2,3) depending on history; just generate and submit.
	r2, err := tr.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	res := []Result{}
	for i, p := range r2.Pairs {
		w := p.A
		if i%2 == 1 {
			w = p.B
		}
		res = append(res, Result{Pair: p, Winner: w})
	}
	if _, err := tr.SubmitResults(res); err != nil {
		t.Fatal(err)
	}
	st := tr.Standings()
	if len(st) != 4 {
		t.Fatalf("standings len = %d", len(st))
	}
	for i, s := range st {
		if s.Rank != i+1 {
			t.Fatalf("rank mismatch: %+v", s)
		}
		if i > 0 {
			prev := st[i-1]
			if prev.Score < s.Score {
				t.Fatal("standings not sorted by score desc")
			}
		}
	}
	// Winners of round 2 have 3 points and must rank above the 1-point rows.
	if st[0].Score != 3 || st[1].Score != 3 {
		t.Fatalf("top scores = %d,%d", st[0].Score, st[1].Score)
	}
}

func TestWithdrawKeepsHistory(t *testing.T) {
	tr := mustNew(t, 1, 2, 3, 4)
	if _, err := tr.GenerateRound(); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.SubmitResults([]Result{
		{Pair: Pair{1, 2}, Winner: 1},
		{Pair: Pair{3, 4}, Winner: 3},
	}); err != nil {
		t.Fatal(err)
	}
	if err := tr.Withdraw(3); err != nil {
		t.Fatal(err)
	}
	r, err := tr.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range r.Pairs {
		if p.A == 3 || p.B == 3 {
			t.Fatalf("withdrawn player paired: %v", r.Pairs)
		}
	}
	if r.Bye == 3 {
		t.Fatal("withdrawn player received bye")
	}
	found := false
	for _, s := range tr.Standings() {
		if s.Player == 3 {
			found = true
			if s.Score != 2 || !s.Withdrawn {
				t.Fatalf("withdrawn standing = %+v", s)
			}
		}
	}
	if !found {
		t.Fatal("withdrawn player missing from standings")
	}
}

func TestSaveLoadConsistency(t *testing.T) {
	dir := t.TempDir()
	tr := mustNew(t, 1, 2, 3, 4, 5)
	if _, err := tr.GenerateRound(); err != nil {
		t.Fatal(err)
	}
	r := tr.currentRound()
	res := []Result{}
	for _, p := range r.Pairs {
		res = append(res, Result{Pair: p, Winner: p.A})
	}
	if _, err := tr.SubmitResults(res); err != nil {
		t.Fatal(err)
	}
	if err := tr.Save(dir); err != nil {
		t.Fatal(err)
	}
	back, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tr.Standings(), back.Standings()) {
		t.Fatal("standings differ after reload")
	}
	// Pairing after reload must be identical to pairing in memory.
	r1, err := tr.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	r2, err := back.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r1, r2) {
		t.Fatalf("rounds differ after reload: %v vs %v", r1, r2)
	}
}

func TestByeScoresTwoAndNotAnOpponent(t *testing.T) {
	tr := mustNew(t, 1, 2, 3)
	r, err := tr.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	if r.Bye != 1 {
		t.Fatalf("bye = %d, want 1 (all zero scores, lowest id)", r.Bye)
	}
	if _, err := tr.SubmitResults([]Result{{Pair: Pair{2, 3}, Winner: 2}}); err != nil {
		t.Fatal(err)
	}
	if tr.Scores[1] != 2 {
		t.Fatalf("bye score = %d, want 2", tr.Scores[1])
	}
	for _, s := range tr.Standings() {
		if s.Player == 1 && len(s.Opponents) != 0 {
			t.Fatalf("bye created phantom opponents: %v", s.Opponents)
		}
	}
}
