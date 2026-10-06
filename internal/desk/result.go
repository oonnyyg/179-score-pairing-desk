package desk

import (
	"errors"
	"fmt"
	"sort"
)

var ErrRoundConfirmed = errors.New("current round already confirmed")

// SubmitResults validates the whole batch against the open round and, only
// if every entry is valid, applies all scores atomically.
func (t *Tournament) SubmitResults(results []Result) (*Round, error) {
	r := t.currentRound()
	if r == nil {
		return nil, errors.New("no round has been generated")
	}
	if r.Confirmed {
		return nil, ErrRoundConfirmed
	}
	want := map[Pair]bool{}
	for _, p := range r.Pairs {
		want[p] = true
	}
	seen := map[Pair]bool{}
	for _, res := range results {
		p := res.Pair
		if p.A > p.B {
			p.A, p.B = p.B, p.A
		}
		if !want[p] {
			return nil, fmt.Errorf("result references unknown or duplicate pairing %d-%d", p.A, p.B)
		}
		if seen[p] {
			return nil, fmt.Errorf("duplicate result for pairing %d-%d", p.A, p.B)
		}
		seen[p] = true
		if res.Winner != 0 && res.Winner != p.A && res.Winner != p.B {
			return nil, fmt.Errorf("invalid winner %d for pairing %d-%d", res.Winner, p.A, p.B)
		}
	}
	if len(seen) != len(want) {
		return nil, fmt.Errorf("missing results: got %d, need %d", len(seen), len(want))
	}
	// All valid: apply.
	for _, res := range results {
		p := res.Pair
		if p.A > p.B {
			p.A, p.B = p.B, p.A
		}
		switch res.Winner {
		case 0:
			t.Scores[p.A]++
			t.Scores[p.B]++
		case p.A:
			t.Scores[p.A] += 2
		default:
			t.Scores[p.B] += 2
		}
		t.Opponents[p.A] = append(t.Opponents[p.A], p.B)
		t.Opponents[p.B] = append(t.Opponents[p.B], p.A)
		r.Results = append(r.Results, Result{Pair: p, Winner: res.Winner})
	}
	if r.Bye != 0 {
		t.Scores[r.Bye] += 2
		t.Byes[r.Bye] = true
	}
	r.Confirmed = true
	return r, nil
}

// Standing is one row of the ranking table.
type Standing struct {
	Rank        int   `json:"rank"`
	Player      int   `json:"player"`
	Score       int   `json:"score"`
	OpponentSum int   `json:"opponentScoreSum"`
	Opponents   []int `json:"opponents"`
	Withdrawn   bool  `json:"withdrawn,omitempty"`
}

// Standings ranks all players (including withdrawn, history preserved) by
// total score desc, then sum of actual opponents' current scores desc
// (byes do not count as opponents), then player id asc.
func (t *Tournament) Standings() []Standing {
	rows := make([]Standing, 0, len(t.Players))
	for _, p := range t.Players {
		sum := 0
		opps := append([]int(nil), t.Opponents[p]...)
		for _, q := range opps {
			sum += t.Scores[q]
		}
		rows = append(rows, Standing{
			Player:      p,
			Score:       t.Scores[p],
			OpponentSum: sum,
			Opponents:   opps,
			Withdrawn:   t.Withdrawn[p],
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.OpponentSum != b.OpponentSum {
			return a.OpponentSum > b.OpponentSum
		}
		return a.Player < b.Player
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}
