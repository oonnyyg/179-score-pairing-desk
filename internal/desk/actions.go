package desk

import (
	"errors"
	"fmt"
	"sort"
)

// ResultInput is one reported game result: players a and b with an outcome.
type ResultInput struct {
	A       int     `json:"a"`
	B       int     `json:"b"`
	Outcome Outcome `json:"outcome"`
}

// SubmitResults validates the whole batch for the pending round and applies
// it atomically. On any validation error nothing is scored.
func (d *Desk) SubmitResults(results []ResultInput) (*Round, error) {
	r := d.pendingRound()
	if r == nil {
		return nil, errors.New("no pending round awaiting results")
	}
	// Index expected games (excluding the bye).
	type key struct{ a, b int }
	expected := map[key]Outcome{}
	for _, g := range r.Games {
		if g.Outcome == OutcomeBye {
			continue
		}
		expected[key{g.A, g.B}] = ""
	}
	seen := map[key]bool{}
	for _, res := range results {
		switch res.Outcome {
		case OutcomeAWin, OutcomeDraw, OutcomeBWin:
		default:
			return nil, fmt.Errorf("invalid outcome %q", res.Outcome)
		}
		a, b := res.A, res.B
		if a > b {
			a, b = b, a
		}
		k := key{a, b}
		if _, ok := expected[k]; !ok {
			return nil, fmt.Errorf("game %d vs %d is not part of round %d", a, b, r.Number)
		}
		if seen[k] {
			return nil, fmt.Errorf("duplicate result for game %d vs %d", a, b)
		}
		seen[k] = true
		expected[k] = res.Outcome
	}
	if len(seen) != len(expected) {
		return nil, fmt.Errorf("missing results: got %d of %d games", len(seen), len(expected))
	}
	// Apply.
	for i, g := range r.Games {
		if g.Outcome == OutcomeBye {
			continue
		}
		r.Games[i].Outcome = expected[key{g.A, g.B}]
	}
	r.Confirmed = true
	return r, nil
}

// Withdraw removes a player from future pairings. History is preserved.
// Only allowed between rounds (no pending unconfirmed round).
func (d *Desk) Withdraw(id int) error {
	p, ok := d.Players[id]
	if !ok {
		return fmt.Errorf("unknown player %d", id)
	}
	if p.Withdrawn {
		return fmt.Errorf("player %d already withdrawn", id)
	}
	if d.pendingRound() != nil {
		return errors.New("cannot withdraw while a round awaits results")
	}
	p.Withdrawn = true
	return nil
}

// Standing is one row of the ranking table.
type Standing struct {
	Rank      int   `json:"rank"`
	ID        int   `json:"id"`
	Score     int   `json:"score"`
	Buchholz  int   `json:"buchholz"` // sum of actual opponents' current scores
	Opponents []int `json:"opponents"`
	Withdrawn bool  `json:"withdrawn"`
}

// Standings ranks by score desc, then Buchholz desc, then id asc.
// Withdrawn players keep their history and are listed last? No: they keep
// their earned rank; ranking rules apply uniformly.
func (d *Desk) Standings() []Standing {
	scores := d.scores()
	opp := d.opponents()
	rows := make([]Standing, 0, len(d.Order))
	for _, id := range d.Order {
		bh := 0
		opps := []int{}
		for o := range opp[id] {
			bh += scores[o]
			opps = append(opps, o)
		}
		sort.Ints(opps)
		rows = append(rows, Standing{
			ID:        id,
			Score:     scores[id],
			Buchholz:  bh,
			Opponents: opps,
			Withdrawn: d.Players[id].Withdrawn,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Score != rows[j].Score {
			return rows[i].Score > rows[j].Score
		}
		if rows[i].Buchholz != rows[j].Buchholz {
			return rows[i].Buchholz > rows[j].Buchholz
		}
		return rows[i].ID < rows[j].ID
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}
