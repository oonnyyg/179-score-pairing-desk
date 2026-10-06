// Command pairdesk reads JSONL operations from stdin and writes one JSON
// response per line to stdout. Tournament state is persisted per directory.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"example.com/score-pairing-desk/internal/desk"
)

type request struct {
	Op      string        `json:"op"`
	Dir     string        `json:"dir"`
	Players []int         `json:"players,omitempty"`
	Player  int           `json:"player,omitempty"`
	Results []desk.Result `json:"results,omitempty"`
}

type response struct {
	OK        bool            `json:"ok"`
	Error     string          `json:"error,omitempty"`
	Round     *desk.Round     `json:"round,omitempty"`
	Rounds    []*desk.Round   `json:"rounds,omitempty"`
	Standings []desk.Standing `json:"standings,omitempty"`
}

func main() {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	enc := json.NewEncoder(os.Stdout)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		enc.Encode(handle(line))
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handle(line []byte) response {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return response{Error: "invalid JSON: " + err.Error()}
	}
	if req.Dir == "" {
		return response{Error: "dir is required"}
	}
	switch req.Op {
	case "init":
		t, err := desk.New(req.Players)
		if err != nil {
			return response{Error: err.Error()}
		}
		if err := t.Save(req.Dir); err != nil {
			return response{Error: err.Error()}
		}
		return response{OK: true}
	case "withdraw":
		t, err := desk.Load(req.Dir)
		if err != nil {
			return response{Error: err.Error()}
		}
		if err := t.Withdraw(req.Player); err != nil {
			return response{Error: err.Error()}
		}
		if err := t.Save(req.Dir); err != nil {
			return response{Error: err.Error()}
		}
		return response{OK: true}
	case "pair":
		t, err := desk.Load(req.Dir)
		if err != nil {
			return response{Error: err.Error()}
		}
		r, err := t.GenerateRound()
		if err != nil {
			return response{Error: err.Error()}
		}
		if err := t.Save(req.Dir); err != nil {
			return response{Error: err.Error()}
		}
		return response{OK: true, Round: r}
	case "submit":
		t, err := desk.Load(req.Dir)
		if err != nil {
			return response{Error: err.Error()}
		}
		r, err := t.SubmitResults(req.Results)
		if err != nil {
			return response{Error: err.Error()}
		}
		if err := t.Save(req.Dir); err != nil {
			return response{Error: err.Error()}
		}
		return response{OK: true, Round: r}
	case "standings":
		t, err := desk.Load(req.Dir)
		if err != nil {
			return response{Error: err.Error()}
		}
		return response{OK: true, Standings: t.Standings()}
	case "rounds":
		t, err := desk.Load(req.Dir)
		if err != nil {
			return response{Error: err.Error()}
		}
		return response{OK: true, Rounds: t.Rounds}
	default:
		return response{Error: "unknown op: " + req.Op}
	}
}
