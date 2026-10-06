package desk

import (
	"reflect"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	d := mustNew(t, 1, 2, 3, 4, 5)
	r1, err := d.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	submitAll(t, d, r1, OutcomeDraw)
	if err := d.Withdraw(5); err != nil {
		t.Fatal(err)
	}
	r2, err := d.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	submitAll(t, d, r2, OutcomeAWin)
	if err := Save(dir, d); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil {
		t.Fatal("expected saved state")
	}
	if !reflect.DeepEqual(d.Standings(), loaded.Standings()) {
		t.Fatal("standings differ after reload")
	}
	if !reflect.DeepEqual(d.Rounds, loaded.Rounds) {
		t.Fatal("round history differs after reload")
	}
	// Pairing continues deterministically from the reloaded state.
	r3a, err := d.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	r3b, err := loaded.GenerateRound()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r3a, r3b) {
		t.Fatal("next pairing differs after reload")
	}
}

func TestLoadMissingDir(t *testing.T) {
	d, err := Load(t.TempDir() + "/nope")
	if err != nil {
		t.Fatal(err)
	}
	if d != nil {
		t.Fatal("expected nil desk for missing state")
	}
}
