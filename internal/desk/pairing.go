package desk

import (
	"errors"
	"sort"
)

// abs of an int.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// chooseBye picks the bye player among active players: prefer players who
// never had a bye, then lowest score, then lowest id. If every active
// player already had a bye, the pool falls back to all active players.
func chooseBye(ids []int, scores map[int]int, players map[int]*Player) int {
	pool := []int{}
	for _, id := range ids {
		if !players[id].HadBye {
			pool = append(pool, id)
		}
	}
	if len(pool) == 0 {
		pool = append(pool, ids...)
	}
	sort.Slice(pool, func(i, j int) bool {
		if scores[pool[i]] != scores[pool[j]] {
			return scores[pool[i]] < scores[pool[j]]
		}
		return pool[i] < pool[j]
	})
	return pool[0]
}

// matching is a set of pairs; each pair has the smaller id first and
// pairs are sorted lexicographically.
type matching [][2]int

func normalize(pairs [][2]int) matching {
	m := make(matching, len(pairs))
	for i, p := range pairs {
		if p[0] > p[1] {
			p[0], p[1] = p[1], p[0]
		}
		m[i] = p
	}
	sort.Slice(m, func(i, j int) bool {
		if m[i][0] != m[j][0] {
			return m[i][0] < m[j][0]
		}
		return m[i][1] < m[j][1]
	})
	return m
}

// less compares two normalized matchings lexicographically.
func (m matching) less(o matching) bool {
	for i := range m {
		if m[i][0] != o[i][0] {
			return m[i][0] < o[i][0]
		}
		if m[i][1] != o[i][1] {
			return m[i][1] < o[i][1]
		}
	}
	return false
}

// bestMatching searches all perfect matchings of ids (even length) that
// avoid rematches, minimizing the sum of |score diff| over pairs. Ties are
// broken by choosing the lexicographically smallest normalized matching.
// Returns the matching and its objective value.
func bestMatching(ids []int, scores map[int]int, past map[int]map[int]bool) (matching, int, error) {
	if len(ids)%2 != 0 {
		return nil, 0, errors.New("odd number of players to pair")
	}
	var best matching
	bestObj := -1
	used := make([]bool, len(ids))
	cur := make([][2]int, 0, len(ids)/2)
	curObj := 0

	var rec func()
	rec = func() {
		// pick first unused player
		i := -1
		for k := range ids {
			if !used[k] {
				i = k
				break
			}
		}
		if i == -1 {
			m := normalize(cur)
			if bestObj == -1 || curObj < bestObj || (curObj == bestObj && m.less(best)) {
				best = m
				bestObj = curObj
			}
			return
		}
		used[i] = true
		for j := i + 1; j < len(ids); j++ {
			if used[j] {
				continue
			}
			a, b := ids[i], ids[j]
			if past[a][b] {
				continue // no rematches
			}
			used[j] = true
			cur = append(cur, [2]int{a, b})
			curObj += abs(scores[a] - scores[b])
			if bestObj == -1 || curObj <= bestObj {
				rec()
			}
			curObj -= abs(scores[a] - scores[b])
			cur = cur[:len(cur)-1]
			used[j] = false
		}
		used[i] = false
	}
	rec()
	if bestObj == -1 {
		return nil, 0, errors.New("no valid pairing exists without rematches")
	}
	return best, bestObj, nil
}

// GenerateRound creates the next round. It fails if the previous round is
// not fully confirmed yet.
func (d *Desk) GenerateRound() (*Round, error) {
	if p := d.pendingRound(); p != nil {
		return nil, errors.New("current round results are not fully submitted")
	}
	ids := d.activeIDs()
	if len(ids) < 2 {
		return nil, errors.New("fewer than two active players")
	}
	scores := d.scores()
	past := d.opponents()

	bye := 0
	if len(ids)%2 == 1 {
		bye = chooseBye(ids, scores, d.Players)
		rest := make([]int, 0, len(ids)-1)
		for _, id := range ids {
			if id != bye {
				rest = append(rest, id)
			}
		}
		ids = rest
	}
	m, obj, err := bestMatching(ids, scores, past)
	if err != nil {
		// Do not change the chosen bye state on failure.
		return nil, err
	}
	r := &Round{Number: len(d.Rounds) + 1, Bye: bye, Objective: obj}
	if bye != 0 {
		r.Games = append(r.Games, Game{A: bye, Outcome: OutcomeBye})
	}
	for _, p := range m {
		r.Games = append(r.Games, Game{A: p[0], B: p[1]})
	}
	d.Rounds = append(d.Rounds, r)
	if bye != 0 {
		d.Players[bye].HadBye = true
	}
	return r, nil
}
