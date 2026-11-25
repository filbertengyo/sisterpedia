// Package store is the server's persistent storage (SQLite). It stores only
// public artifacts (parameters, public key, relinearization key) and
// ciphertexts (balance, transaction metadata and totals), as binary BLOBs.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

var (
	ErrNotFound = errors.New("account not found")
	ErrExists   = errors.New("account already exists")
)

// Account is a stored account. All byte fields are library-serialized blobs.
type Account struct {
	UserID     string
	Profile    string
	Params     []byte
	PublicKey  []byte
	RelinKey   []byte
	EncBalance []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Transaction is a stored history entry. Product/discount codes are not
// stored in the clear; they only exist inside the encrypted metadata.
type Transaction struct {
	ID          string
	UserID      string
	Seq         int64
	EncMetadata []byte
	EncTotal    []byte
	CreatedAt   time.Time
}

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS accounts (
	user_id     TEXT PRIMARY KEY,
	profile     TEXT NOT NULL,
	params      BLOB NOT NULL,
	public_key  BLOB NOT NULL,
	relin_key   BLOB NOT NULL,
	enc_balance BLOB NOT NULL,
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS transactions (
	transaction_id TEXT PRIMARY KEY,
	user_id        TEXT NOT NULL REFERENCES accounts(user_id),
	seq            INTEGER NOT NULL,
	enc_metadata   BLOB NOT NULL,
	enc_total      BLOB NOT NULL,
	created_at     TEXT NOT NULL,
	UNIQUE (user_id, seq)
);
CREATE INDEX IF NOT EXISTS idx_transactions_user ON transactions(user_id, seq);
`

// SchemaVersion is the current database schema version.
const SchemaVersion = 1

// Open opens (creating if needed) the database at path and applies the schema.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(FULL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// One connection: SQLite serializes writers anyway; this avoids SQLITE_BUSY.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_version`).Scan(&n); err != nil {
		_ = db.Close()
		return nil, err
	}
	if n == 0 {
		if _, err := db.Exec(`INSERT INTO schema_version(version) VALUES (?)`, SchemaVersion); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return &Store{db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTS(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

// Exists reports whether an account exists.
func (s *Store) Exists(ctx context.Context, userID string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM accounts WHERE user_id = ?`, userID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// CreateAccount inserts a new account; ErrExists if the user_id is taken.
func (s *Store) CreateAccount(ctx context.Context, a Account) error {
	now := ts(a.CreatedAt)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO accounts(user_id, profile, params, public_key, relin_key, enc_balance, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.UserID, a.Profile, a.Params, a.PublicKey, a.RelinKey, a.EncBalance, now, now)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "unique") {
		return ErrExists
	}
	return err
}

// GetAccount loads an account with its keys and balance.
func (s *Store) GetAccount(ctx context.Context, userID string) (Account, error) {
	var a Account
	var created, updated string
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, profile, params, public_key, relin_key, enc_balance, created_at, updated_at
		 FROM accounts WHERE user_id = ?`, userID).
		Scan(&a.UserID, &a.Profile, &a.Params, &a.PublicKey, &a.RelinKey, &a.EncBalance, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, err
	}
	a.CreatedAt, a.UpdatedAt = parseTS(created), parseTS(updated)
	return a, nil
}

// GetBalance returns the stored balance ciphertext.
func (s *Store) GetBalance(ctx context.Context, userID string) ([]byte, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT enc_balance FROM accounts WHERE user_id = ?`, userID).Scan(&b)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}

// UpdateBalance replaces the balance ciphertext.
func (s *Store) UpdateBalance(ctx context.Context, userID string, enc []byte, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE accounts SET enc_balance = ?, updated_at = ? WHERE user_id = ?`, enc, ts(at), userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ApplyTransaction stores the new balance and appends the history entry in a
// single SQL transaction: either both are persisted or neither is.
func (s *Store) ApplyTransaction(ctx context.Context, userID string, newBalance []byte, t Transaction) (Transaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Transaction{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `UPDATE accounts SET enc_balance = ?, updated_at = ? WHERE user_id = ?`, newBalance, ts(t.CreatedAt), userID)
	if err != nil {
		return Transaction{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Transaction{}, ErrNotFound
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq), 0) + 1 FROM transactions WHERE user_id = ?`, userID).Scan(&t.Seq); err != nil {
		return Transaction{}, err
	}
	t.UserID = userID
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO transactions(transaction_id, user_id, seq, enc_metadata, enc_total, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, userID, t.Seq, t.EncMetadata, t.EncTotal, ts(t.CreatedAt)); err != nil {
		return Transaction{}, err
	}
	if err := tx.Commit(); err != nil {
		return Transaction{}, err
	}
	return t, nil
}

// ListTransactions returns the history from oldest to newest.
func (s *Store) ListTransactions(ctx context.Context, userID string) ([]Transaction, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT transaction_id, user_id, seq, enc_metadata, enc_total, created_at
		 FROM transactions WHERE user_id = ? ORDER BY seq ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Transaction{}
	for rows.Next() {
		var t Transaction
		var created string
		if err := rows.Scan(&t.ID, &t.UserID, &t.Seq, &t.EncMetadata, &t.EncTotal, &created); err != nil {
			return nil, err
		}
		t.CreatedAt = parseTS(created)
		out = append(out, t)
	}
	return out, rows.Err()
}
