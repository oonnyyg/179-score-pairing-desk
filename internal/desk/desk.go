// Package desk implements a custom score-based pairing tournament manager.
// It is NOT a full implementation of any official Swiss-system regulation.
package desk

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const MaxPlayers = 16

// Pair is one game in a round, A < B.
type Pair struct {
	A int `json:"a"`
	B int `json:"b"`
}

// Round records a generated round and, once confirmed, its results.
type Round struct {
	Number    int      `json:"number"`
	Pairs     []Pair   `json:"pairs"`
	Bye       int      `json:"bye"` // 0 means no bye
	Objective int      `json:"objective"`
	Results   []Result `json:"results,omitempty"`
	Confirmed bool     `json:"confirmed"`
}

// Result is the outcome of one game: Winner is a player id, or 0 for a draw.
type Result struct {
	Pair   Pair `json:"pair"`
	Winner int  `json:"winner"` // 0 = draw
}

// Tournament is the full persistent state.
type Tournament struct {
	Players   []int         `json:"players"`
	Withdrawn map[int]bool  `json:"withdrawn,omitempty"`
	Scores    map[int]int   `json:"scores"`
	Byes      map[int]bool  `json:"byes,omitempty"`
	Opponents map[int][]int `json:"opponents"`
	Rounds    []*Round      `json:"rounds"`
}

// New creates a tournament with the given players.
func New(players []int) (*Tournament, error) {
	if len(players) == 0 || len(players) > MaxPlayers {
		return nil, fmt.Errorf("player count must be 1..%d", MaxPlayers)
	}
	seen := map[int]bool{}
	for _, p := range players {
		if p <= 0 {
			return nil, fmt.Errorf("player id %d is not a positive integer", p)
		}
		if seen[p] {
			return nil, fmt.Errorf("duplicate player id %d", p)
		}
		seen[p] = true
	}
	t := &Tournament{
		Players:   append([]int(nil), players...),
		Withdrawn: map[int]bool{},
		Scores:    map[int]int{},
		Byes:      map[int]bool{},
		Opponents: map[int][]int{},
	}
	sort.Ints(t.Players)
	return t, nil
}

const stateFile = "tournament.json"

// Save writes the tournament to dir/tournament.json.
func (t *Tournament) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, stateFile), data, 0o644)
}

// Load reads a tournament from dir/tournament.json.
func Load(dir string) (*Tournament, error) {
	data, err := os.ReadFile(filepath.Join(dir, stateFile))
	if err != nil {
		return nil, err
	}
	t := &Tournament{}
	if err := json.Unmarshal(data, t); err != nil {
		return nil, err
	}
	if t.Withdrawn == nil {
		t.Withdrawn = map[int]bool{}
	}
	if t.Scores == nil {
		t.Scores = map[int]int{}
	}
	if t.Byes == nil {
		t.Byes = map[int]bool{}
	}
	if t.Opponents == nil {
		t.Opponents = map[int][]int{}
	}
	return t, nil
}

var ErrRoundOpen = errors.New("current round results not fully submitted")
var ErrNoPairing = errors.New("no valid complete pairing exists")

func (t *Tournament) active() []int {
	out := []int{}
	for _, p := range t.Players {
		if !t.Withdrawn[p] {
			out = append(out, p)
		}
	}
	return out
}

func (t *Tournament) currentRound() *Round {
	if len(t.Rounds) == 0 {
		return nil
	}
	return t.Rounds[len(t.Rounds)-1]
}

// Withdraw removes a player from future pairings; history is kept.
func (t *Tournament) Withdraw(player int) error {
	if !t.isPlayer(player) {
		return fmt.Errorf("unknown player %d", player)
	}
	if t.Withdrawn[player] {
		return fmt.Errorf("player %d already withdrawn", player)
	}
	t.Withdrawn[player] = true
	return nil
}

func (t *Tournament) isPlayer(p int) bool {
	for _, q := range t.Players {
		if q == p {
			return true
		}
	}
	return false
}
