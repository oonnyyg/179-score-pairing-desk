// Package desk implements a custom score-based pairing tournament manager.
// It is NOT a full implementation of any official Swiss-system regulation.
package desk

import (
	"errors"
	"fmt"
	"sort"
)

// Outcome of a single game.
type Outcome string

const (
	OutcomeAWin Outcome = "a_win"
	OutcomeDraw Outcome = "draw"
	OutcomeBWin Outcome = "b_win"
	OutcomeBye  Outcome = "bye"
)

// Game is one pairing in a round. A and B are player numbers.
// For a bye, B is 0 and Outcome is OutcomeBye.
type Game struct {
	A       int     `json:"a"`
	B       int     `json:"b,omitempty"`
	Outcome Outcome `json:"outcome,omitempty"`
}

// Round records a generated round. Results are filled on submission.
type Round struct {
	Number    int    `json:"number"`
	Games     []Game `json:"games"`
	Bye       int    `json:"bye,omitempty"`
	Objective int    `json:"objective"` // minimized sum of |score diff| over pairs
	Confirmed bool   `json:"confirmed"`
}

// Player is a registered participant.
type Player struct {
	ID        int  `json:"id"`
	Withdrawn bool `json:"withdrawn"`
	HadBye    bool `json:"had_bye"`
}

// Desk is the tournament state.
type Desk struct {
	Players map[int]*Player `json:"players"`
	Order   []int           `json:"order"` // registration order, ascending ids
	Rounds  []*Round        `json:"rounds"`
}

// New creates an empty desk with the given player numbers.
func New(ids []int) (*Desk, error) {
	if len(ids) == 0 {
		return nil, errors.New("player list is empty")
	}
	if len(ids) > 16 {
		return nil, errors.New("at most 16 players allowed")
	}
	d := &Desk{Players: map[int]*Player{}}
	seen := map[int]bool{}
	for _, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("player id %d is not a positive integer", id)
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate player id %d", id)
		}
		seen[id] = true
		d.Players[id] = &Player{ID: id}
		d.Order = append(d.Order, id)
	}
	sort.Ints(d.Order)
	return d, nil
}

// activeIDs returns non-withdrawn players in ascending id order.
func (d *Desk) activeIDs() []int {
	var ids []int
	for _, id := range d.Order {
		if !d.Players[id].Withdrawn {
			ids = append(ids, id)
		}
	}
	return ids
}

// pendingRound returns the current unconfirmed round, or nil.
func (d *Desk) pendingRound() *Round {
	if len(d.Rounds) == 0 {
		return nil
	}
	r := d.Rounds[len(d.Rounds)-1]
	if r.Confirmed {
		return nil
	}
	return r
}

// scores computes current total score per player (all players, incl. withdrawn).
func (d *Desk) scores() map[int]int {
	s := map[int]int{}
	for id := range d.Players {
		s[id] = 0
	}
	for _, r := range d.Rounds {
		if !r.Confirmed {
			continue
		}
		for _, g := range r.Games {
			switch g.Outcome {
			case OutcomeAWin:
				s[g.A] += 2
			case OutcomeBWin:
				s[g.B] += 2
			case OutcomeDraw:
				s[g.A]++
				s[g.B]++
			case OutcomeBye:
				s[g.A] += 2
			}
		}
	}
	return s
}

// opponents returns the set of actual opponents each player has faced
// in confirmed rounds. Byes create no opponent.
func (d *Desk) opponents() map[int]map[int]bool {
	opp := map[int]map[int]bool{}
	for id := range d.Players {
		opp[id] = map[int]bool{}
	}
	for _, r := range d.Rounds {
		if !r.Confirmed {
			continue
		}
		for _, g := range r.Games {
			if g.Outcome == OutcomeBye {
				continue
			}
			opp[g.A][g.B] = true
			opp[g.B][g.A] = true
		}
	}
	return opp
}
