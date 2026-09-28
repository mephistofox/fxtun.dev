package database

import (
	"testing"
	"time"
)

// Traffic totals were reported as zero for every user: SUM over a bigint column
// returns numeric, which reached the repository as pgtype.Numeric behind an
// interface{}, so the int64 type assertion never matched and the zero value was
// returned instead.
func TestGetStats_ReturnsActualByteTotals(t *testing.T) {
	db := newTestDB(t)
	userID := insertUser(t, db, "+70000004242")

	entries := []*UserHistoryEntry{
		{TunnelType: "http", LocalPort: 3000, ConnectedAt: time.Now(), BytesSent: 1500, BytesReceived: 2500},
		{TunnelType: "http", LocalPort: 3001, ConnectedAt: time.Now(), BytesSent: 500, BytesReceived: 1000},
	}
	if err := db.UserHistory.AddBulk(userID, entries); err != nil {
		t.Fatalf("AddBulk: %v", err)
	}

	stats, err := db.UserHistory.GetStats(userID)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalConnections != 2 {
		t.Errorf("TotalConnections = %d, want 2", stats.TotalConnections)
	}
	if stats.TotalBytesSent != 2000 {
		t.Errorf("TotalBytesSent = %d, want 2000", stats.TotalBytesSent)
	}
	if stats.TotalBytesReceived != 3500 {
		t.Errorf("TotalBytesReceived = %d, want 3500", stats.TotalBytesReceived)
	}
}

// Invoice numbers used to come from MAX(invoice_id)+1, so two checkouts racing
// each other received the same number and the loser hit the UNIQUE constraint —
// a failed payment for a user who did nothing wrong.
func TestGetNextInvoiceID_Unique(t *testing.T) {
	db := newTestDB(t)

	const n = 20
	seen := make(chan int64, n)
	for i := 0; i < n; i++ {
		go func() {
			id, err := db.Payments.GetNextInvoiceID()
			if err != nil {
				seen <- 0
				return
			}
			seen <- id
		}()
	}

	unique := make(map[int64]bool, n)
	for i := 0; i < n; i++ {
		id := <-seen
		if id == 0 {
			t.Fatal("GetNextInvoiceID failed")
		}
		if unique[id] {
			t.Fatalf("invoice id %d handed out twice", id)
		}
		unique[id] = true
	}
}
