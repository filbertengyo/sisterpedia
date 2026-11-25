package store

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestAccountAndAtomicTransaction(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "shop.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	a := Account{UserID: "alice", Profile: "p1", Params: []byte("p"), PublicKey: []byte("pk"), RelinKey: []byte("rk"), EncBalance: []byte("b0"), CreatedAt: now}
	if err := s.CreateAccount(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAccount(ctx, a); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := s.ApplyTransaction(ctx, "alice", []byte("b1"), Transaction{ID: "T1", EncMetadata: []byte("m"), EncTotal: []byte("t"), CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	// a failing insert (duplicate transaction id) must roll back the balance update
	if _, err := s.ApplyTransaction(ctx, "alice", []byte("b2"), Transaction{ID: "T1", EncMetadata: []byte("m"), EncTotal: []byte("t"), CreatedAt: now}); err == nil {
		t.Fatal("duplicate transaction id accepted")
	}
	if b, _ := s.GetBalance(ctx, "alice"); !bytes.Equal(b, []byte("b1")) {
		t.Fatalf("balance = %q after rolled back transaction, want b1", b)
	}
	if _, err := s.ApplyTransaction(ctx, "nobody", []byte("x"), Transaction{ID: "T9", CreatedAt: now, EncMetadata: []byte("m"), EncTotal: []byte("t")}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown account: %v", err)
	}
	_ = s.Close()

	// persistence across restarts
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	txs, err := s.ListTransactions(ctx, "alice")
	if err != nil || len(txs) != 1 || txs[0].ID != "T1" || txs[0].Seq != 1 {
		t.Fatalf("history after reopen: %+v %v", txs, err)
	}
	if acc, err := s.GetAccount(ctx, "alice"); err != nil || acc.Profile != "p1" || !bytes.Equal(acc.EncBalance, []byte("b1")) {
		t.Fatalf("account after reopen: %+v %v", acc, err)
	}
}
