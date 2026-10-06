package desk

import "sort"

// GenerateRound computes the next round's bye (if odd) and the optimal
// complete pairing, then records it as the current open round.
func (t *Tournament) GenerateRound() (*Round, error) {
	if r := t.currentRound(); r != nil && !r.Confirmed {
		return nil, ErrRoundOpen
	}
	players := t.active()
	if len(players) < 2 {
		return nil, ErrNoPairing
	}
	bye := 0
	if len(players)%2 == 1 {
		bye = t.pickBye(players)
		rest := make([]int, 0, len(players)-1)
		for _, p := range players {
			if p != bye {
				rest = append(rest, p)
			}
		}
		players = rest
	}
	pairs, obj, ok := t.bestMatching(players)
	if !ok {
		return nil, ErrNoPairing
	}
	r := &Round{
		Number:    len(t.Rounds) + 1,
		Pairs:     pairs,
		Bye:       bye,
		Objective: obj,
	}
	t.Rounds = append(t.Rounds, r)
	return r, nil
}

// pickBye selects the bye player: among active players who never had a bye,
// lowest score first, then lowest id. Falls back to all active players if
// everyone has had a bye already.
func (t *Tournament) pickBye(players []int) int {
	pool := []int{}
	for _, p := range players {
		if !t.Byes[p] {
			pool = append(pool, p)
		}
	}
	if len(pool) == 0 {
		pool = players
	}
	best := pool[0]
	for _, p := range pool[1:] {
		if t.Scores[p] < t.Scores[best] ||
			(t.Scores[p] == t.Scores[best] && p < best) {
			best = p
		}
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// bestMatching exhaustively searches all complete pairings of players
// (len must be even and <= 16), forbidding repeated opponents, minimizing
// the sum of absolute score differences. Ties are broken by choosing the
// lexicographically smallest normalized pairing list.
func (t *Tournament) bestMatching(players []int) ([]Pair, int, bool) {
	sort.Ints(players)
	played := map[[2]int]bool{}
	for p, opps := range t.Opponents {
		for _, q := range opps {
			a, b := p, q
			if a > b {
				a, b = b, a
			}
			played[[2]int{a, b}] = true
		}
	}
	var best []Pair
	bestObj := -1
	cur := make([]Pair, 0, len(players)/2)
	var search func(rest []int, obj int)
	search = func(rest []int, obj int) {
		if bestObj >= 0 && obj > bestObj {
			return
		}
		if len(rest) == 0 {
			if bestObj < 0 || obj < bestObj || lessPairs(cur, best) {
				bestObj = obj
				best = append([]Pair(nil), cur...)
			}
			return
		}
		a := rest[0]
		for i := 1; i < len(rest); i++ {
			b := rest[i]
			if played[[2]int{a, b}] {
				continue
			}
			next := make([]int, 0, len(rest)-2)
			next = append(next, rest[1:i]...)
			next = append(next, rest[i+1:]...)
			cur = append(cur, Pair{A: a, B: b})
			search(next, obj+abs(t.Scores[a]-t.Scores[b]))
			cur = cur[:len(cur)-1]
		}
	}
	search(players, 0)
	if bestObj < 0 {
		return nil, 0, false
	}
	return best, bestObj, true
}

// lessPairs compares two normalized pairing lists lexicographically.
func lessPairs(x, y []Pair) bool {
	for i := range x {
		if x[i].A != y[i].A {
			return x[i].A < y[i].A
		}
		if x[i].B != y[i].B {
			return x[i].B < y[i].B
		}
	}
	return false
}
