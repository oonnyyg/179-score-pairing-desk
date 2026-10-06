// pairdesk is a local score-pairing tournament manager driven by JSONL.
// It reads one JSON operation per line on stdin and writes one JSON
// response per line on stdout. State persists in the directory given by
// -dir, so restarting the process keeps all results.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"example.com/score-pairing-desk/internal/desk"
)

type request struct {
	Op      string             `json:"op"`
	Players []int              `json:"players,omitempty"`
	Player  int                `json:"player,omitempty"`
	Results []desk.ResultInput `json:"results,omitempty"`
}

type response struct {
	OK        bool            `json:"ok"`
	Error     string          `json:"error,omitempty"`
	Round     *desk.Round     `json:"round,omitempty"`
	Standings []desk.Standing `json:"standings,omitempty"`
	Rounds    []*desk.Round   `json:"rounds,omitempty"`
}

func fail(err error) response {
	return response{OK: false, Error: err.Error()}
}

func main() {
	dir := flag.String("dir", "pairdesk-data", "directory for persistent state")
	flag.Parse()

	d, err := desk.Load(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load state:", err)
		os.Exit(1)
	}

	save := func() {
		if err := desk.Save(*dir, d); err != nil {
			fmt.Fprintln(os.Stderr, "save state:", err)
			os.Exit(1)
		}
	}

	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	enc := json.NewEncoder(os.Stdout)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		var resp response
		if err := json.Unmarshal(line, &req); err != nil {
			resp = fail(fmt.Errorf("bad json: %w", err))
			enc.Encode(resp)
			continue
		}
		switch req.Op {
		case "init":
			if d != nil {
				resp = fail(fmt.Errorf("already initialized"))
				break
			}
			nd, err := desk.New(req.Players)
			if err != nil {
				resp = fail(err)
				break
			}
			d = nd
			save()
			resp = response{OK: true}
		case "pair":
			if d == nil {
				resp = fail(fmt.Errorf("not initialized"))
				break
			}
			r, err := d.GenerateRound()
			if err != nil {
				resp = fail(err)
				break
			}
			save()
			resp = response{OK: true, Round: r}
		case "result":
			if d == nil {
				resp = fail(fmt.Errorf("not initialized"))
				break
			}
			r, err := d.SubmitResults(req.Results)
			if err != nil {
				resp = fail(err)
				break
			}
			save()
			resp = response{OK: true, Round: r}
		case "withdraw":
			if d == nil {
				resp = fail(fmt.Errorf("not initialized"))
				break
			}
			if err := d.Withdraw(req.Player); err != nil {
				resp = fail(err)
				break
			}
			save()
			resp = response{OK: true}
		case "standings":
			if d == nil {
				resp = fail(fmt.Errorf("not initialized"))
				break
			}
			resp = response{OK: true, Standings: d.Standings()}
		case "history":
			if d == nil {
				resp = fail(fmt.Errorf("not initialized"))
				break
			}
			resp = response{OK: true, Rounds: d.Rounds}
		default:
			resp = fail(fmt.Errorf("unknown op %q", req.Op))
		}
		enc.Encode(resp)
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "read stdin:", err)
		os.Exit(1)
	}
}
