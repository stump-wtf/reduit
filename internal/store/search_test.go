package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func seedSearchCorpus(t *testing.T, st *Store) {
	t.Helper()
	ctx := context.Background()
	seedMailbox(t, st, testMailboxID, "joe@example.com")
	msgs := []MessageRow{
		{MailboxID: testMailboxID, ProtonID: "p1", Timestamp: time.Date(2026, 7, 4, 7, 0, 0, 0, time.UTC),
			Sender: "billing@shop.example", Subject: "Your receipt #4432", Body: "Thanks for your order. Total $42.00.", Folder: "Inbox"},
		{MailboxID: testMailboxID, ProtonID: "p2", Timestamp: time.Date(2026, 7, 3, 9, 0, 0, 0, time.UTC),
			Sender: "office@school.example", Subject: "Field trip permission", Body: "Please sign the permission slip by Friday.", Folder: "Inbox"},
		{MailboxID: testMailboxID, ProtonID: "p3", Timestamp: time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC),
			Sender: "no-reply@air.example", Subject: "Flight confirmation", Body: "Your itinerary for LH421 is attached.", Folder: "Travel"},
	}
	for _, m := range msgs {
		if _, err := st.UpsertMessage(ctx, m); err != nil {
			t.Fatalf("seed message: %v", err)
		}
	}
}

func TestSearchMessages_KeywordAndPrefix(t *testing.T) {
	st := newTestStore(t)
	seedSearchCorpus(t, st)
	ctx := context.Background()

	// Exact term in subject.
	hits, err := st.SearchMessages(ctx, "receipt", 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 || hits[0].Subject != "Your receipt #4432" {
		t.Fatalf("receipt search: %+v", hits)
	}
	// Prefix: "recei" should still find "receipt".
	if hits, _ := st.SearchMessages(ctx, "recei", 0); len(hits) != 1 {
		t.Errorf("prefix search 'recei' found %d, want 1", len(hits))
	}
	// Body term.
	if hits, _ := st.SearchMessages(ctx, "permission", 0); len(hits) != 1 {
		t.Errorf("body search 'permission' found %d, want 1", len(hits))
	}
	// Multi-term AND: both words must be present (in the same message).
	if hits, _ := st.SearchMessages(ctx, "permission slip", 0); len(hits) != 1 {
		t.Errorf("AND search found %d, want 1", len(hits))
	}
	if hits, _ := st.SearchMessages(ctx, "receipt permission", 0); len(hits) != 0 {
		t.Errorf("AND across messages should find 0, got %d", len(hits))
	}
}

func TestSearchMessages_SnippetHighlights(t *testing.T) {
	st := newTestStore(t)
	seedSearchCorpus(t, st)
	hits, _ := st.SearchMessages(context.Background(), "itinerary", 0)
	if len(hits) != 1 {
		t.Fatalf("want 1 hit, got %d", len(hits))
	}
	if !containsBracket(hits[0].Snippet) {
		t.Errorf("snippet should bracket the match: %q", hits[0].Snippet)
	}
}

func containsBracket(s string) bool {
	return len(s) > 0 && (indexByte(s, '[') >= 0)
}
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func TestSearchMessages_HostileQueryNeverErrors(t *testing.T) {
	st := newTestStore(t)
	seedSearchCorpus(t, st)
	ctx := context.Background()
	// FTS5 operator characters / malformed expressions must degrade to "no
	// matches", never error (SPEC-0008 "always available").
	for _, q := range []string{`"`, `(`, `receipt AND`, `* * *`, `NEAR(`, `col:`, `^`, `-`, `""`, ``, `   `,
		"\x00null", "recei\x00pt", "a\x00", "col: :", "\x1b[31m"} {
		hits, err := st.SearchMessages(ctx, q, 0)
		if err != nil {
			t.Errorf("query %q errored: %v (should degrade to no matches)", q, err)
		}
		_ = hits
	}
}

func TestGetMessage(t *testing.T) {
	st := newTestStore(t)
	seedSearchCorpus(t, st)
	ctx := context.Background()
	hits, _ := st.SearchMessages(ctx, "receipt", 0)
	if len(hits) != 1 {
		t.Fatalf("setup: want 1 hit")
	}
	d, found, err := st.GetMessage(ctx, hits[0].Hash)
	if err != nil || !found {
		t.Fatalf("GetMessage: found=%v err=%v", found, err)
	}
	if d.Subject != "Your receipt #4432" || d.Body == "" {
		t.Errorf("message detail wrong: %+v", d)
	}
	// Unknown hash: not found, no error.
	if _, found, err := st.GetMessage(ctx, "nope"); found || err != nil {
		t.Errorf("unknown hash: found=%v err=%v", found, err)
	}
}

func TestSearchMessages_EmptyCache(t *testing.T) {
	st := newTestStore(t)
	seedMailbox(t, st, testMailboxID, "joe@example.com")
	if hits, err := st.SearchMessages(context.Background(), "anything", 0); err != nil || len(hits) != 0 {
		t.Errorf("empty-cache search: hits=%d err=%v", len(hits), err)
	}
}

// TestIsFTSSyntaxError_Classification pins the narrowed classifier against the
// real driver's error strings (captured by probe, issue #234): genuine
// user-input syntax errors are swallowed, schema/system regressions are NOT.
func TestIsFTSSyntaxError_Classification(t *testing.T) {
	syntax := []string{
		"SQL logic error: unterminated string (1)",          // bare `"`
		"SQL logic error: fts5: syntax error near \"\" (1)", // dangling operator / group
		"SQL logic error: malformed MATCH expression (1)",
	}
	for _, s := range syntax {
		if !isFTSSyntaxError(errors.New(s)) {
			t.Errorf("isFTSSyntaxError(%q) = false, want true (user syntax error)", s)
		}
	}
	regressions := []string{
		"SQL logic error: no such column: foo (1)",         // renamed/dropped column
		"SQL logic error: no such table: messages_fts (1)", // dropped FTS table
		"SQL logic error: unknown special query:  (1)",     // not a query-syntax class
		"SQL logic error: no such column: nosuchcol (1)",
	}
	for _, s := range regressions {
		if isFTSSyntaxError(errors.New(s)) {
			t.Errorf("isFTSSyntaxError(%q) = true, want false (schema regression must surface)", s)
		}
	}
	if isFTSSyntaxError(nil) {
		t.Error("isFTSSyntaxError(nil) = true, want false")
	}
}

// TestSearchMessages_BrokenFTSSchemaErrors proves a schema regression (here a
// dropped FTS table) surfaces as an ERROR, not silently as "no matches"
// (issue #234).
func TestSearchMessages_BrokenFTSSchemaErrors(t *testing.T) {
	st := newTestStore(t)
	seedSearchCorpus(t, st)
	if _, err := st.DB.Exec(`DROP TABLE messages_fts`); err != nil {
		t.Fatalf("drop fts table: %v", err)
	}
	if _, err := st.SearchMessages(context.Background(), "receipt", 0); err == nil {
		t.Error("SearchMessages with dropped messages_fts = nil error, want a schema error (regression must not masquerade as no-matches)")
	}
}
