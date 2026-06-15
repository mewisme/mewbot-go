package wallet

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mewisme/mewbot-go/internal/store"
)

const walletFile = "wallet.json"

// walletLock serializes read-modify-write cycles on the wallet file.
var walletLock sync.Mutex

// userWallet is a single user's balance entry.
type userWallet struct {
	Balance   uint64 `json:"balance"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// walletData is the on-disk wallet document.
type walletData struct {
	Users map[string]*userWallet `json:"users"`
	Unit  string                 `json:"unit"`
}

func newWalletData() *walletData {
	return &walletData{Users: map[string]*userWallet{}, Unit: ""}
}

func (w *walletData) hasUser(userID string) bool {
	_, ok := w.Users[userID]
	return ok
}

func (w *walletData) balanceIfExists(userID string) (uint64, bool) {
	if u, ok := w.Users[userID]; ok {
		return u.Balance, true
	}
	return 0, false
}

func (w *walletData) addBalance(userID string, amount int64, now string) uint64 {
	amt := clampNonNeg(amount)
	u := w.Users[userID]
	if u == nil {
		u = &userWallet{}
		w.Users[userID] = u
	}
	u.Balance += amt
	u.UpdatedAt = now
	return u.Balance
}

func (w *walletData) subtractBalance(userID string, amount int64, now string) (uint64, error) {
	amt := clampNonNeg(amount)
	u := w.Users[userID]
	if u == nil {
		u = &userWallet{}
		w.Users[userID] = u
	}
	if u.Balance < amt {
		return 0, fmt.Errorf("insufficient balance")
	}
	u.Balance -= amt
	u.UpdatedAt = now
	return u.Balance, nil
}

// initUser sets a balance unconditionally (used by reset).
func (w *walletData) initUser(userID string, balance int64, now string) {
	w.Users[userID] = &userWallet{Balance: clampNonNeg(balance), UpdatedAt: now}
}

// initUserIfNew creates a wallet only when absent. Returns true if created.
func (w *walletData) initUserIfNew(userID string, balance int64, now string) bool {
	if _, ok := w.Users[userID]; ok {
		return false
	}
	w.Users[userID] = &userWallet{Balance: clampNonNeg(balance), UpdatedAt: now}
	return true
}

func clampNonNeg(n int64) uint64 {
	if n < 0 {
		return 0
	}
	return uint64(n)
}

func loadWallet() *walletData {
	raw, err := store.Load(walletFile)
	if err != nil || strings.TrimSpace(raw) == "" {
		return newWalletData()
	}
	var d walletData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return newWalletData()
	}
	if d.Users == nil {
		d.Users = map[string]*userWallet{}
	}
	return &d
}

func saveWallet(d *walletData) error {
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return store.Save(walletFile, string(b))
}

func nowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// formatNumber adds thousands separators to a number.
func formatNumber(n uint64) string {
	s := strconv.FormatUint(n, 10)
	var b strings.Builder
	l := len(s)
	for i, c := range s {
		if i > 0 && (l-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// formatBalance renders a balance with the optional unit suffix.
func formatBalance(bal uint64, unit string) string {
	num := formatNumber(bal)
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return num
	}
	return num + " " + unit
}
