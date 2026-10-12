package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// scriptedGitHub serves canned upstream pages keyed by "path|page" and
// records every query it receives for propagation assertions.
type scriptedGitHub struct {
	pages    map[string]string
	statuses map[string]int
	queries  []string
	calls    int
}

func (s *scriptedGitHub) handler(w http.ResponseWriter, r *http.Request) {
	s.calls++
	s.queries = append(s.queries, r.URL.Path+"?"+r.URL.RawQuery)
	key := r.URL.Path + "|page=" + r.URL.Query().Get("page")
	status := s.statuses[key]
	if status == 0 {
		status = http.StatusOK
	}
	body := s.pages[key]
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (s *scriptedGitHub) serve(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(s.handler))
}

func rawIssue(n int, state string, pr bool, updated string, labels string) string {
	prField := ""
	if pr {
		prField = `,"pull_request":{"url":"https://api.github.com/x"}`
	}
	return fmt.Sprintf(`{"number":%d,"title":"Issue %d","body":"body %d","state":%q,"labels":[%s],"created_at":"2024-01-01T00:00:00Z","updated_at":%q,"html_url":"https://github.com/o/r/issues/%d","user":{"login":"alice"}%s}`,
		n, n, n, state, labels, updated, n, prField)
}

func rawPR(n int) string {
	return rawIssue(n, "open", true, "2024-01-02T00:00:00Z", "")
}

func searchEnvelope(items ...string) string {
	return fmt.Sprintf(`{"total_count":%d,"incomplete_results":false,"items":[%s]}`,
		len(items), strings.Join(items, ","))
}

func issueNumbers(issues []Issue) []int {
	out := make([]int, 0, len(issues))
	for _, issue := range issues {
		out = append(out, issue.Number)
	}
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func queriesFor(s *scriptedGitHub, path string) []string {
	var out []string
	for _, q := range s.queries {
		if strings.HasPrefix(q, path+"?") || q == path {
			out = append(out, q)
		}
	}
	return out
}

// padToFull extends an upstream page to issueUpstreamPerPage items with filler
// pull requests. The collector treats a short upstream page as exhaustion
// proof, so pages that must be followed by another page are padded full.
func padToFull(items []string, startPR int) []string {
	n := startPR
	for len(items) < issueUpstreamPerPage {
		items = append(items, rawPR(n))
		n++
	}
	return items
}

// Upstream: page 1 = 8 PRs + issues 1-2 (padded full); page 2 = issues 3-12
// (padded full); page 3 = empty.
func prHeavyFixture() *scriptedGitHub {
	p1 := make([]string, 0, 10)
	for n := 900; n < 908; n++ {
		p1 = append(p1, rawPR(n))
	}
	p1 = append(p1, rawIssue(1, "open", false, "2024-02-01T00:00:00Z", `{"name":"bug"}`))
	p1 = append(p1, rawIssue(2, "open", false, "2024-02-02T00:00:00Z", ""))
	p1 = padToFull(p1, 2000)
	p2 := make([]string, 0, 10)
	for n := 3; n <= 12; n++ {
		p2 = append(p2, rawIssue(n, "open", false, "2024-02-03T00:00:00Z", ""))
	}
	p2 = padToFull(p2, 3000)
	return &scriptedGitHub{
		pages: map[string]string{
			"/repos/o/r/issues|page=1": "[" + strings.Join(p1, ",") + "]",
			"/repos/o/r/issues|page=2": "[" + strings.Join(p2, ",") + "]",
			"/repos/o/r/issues|page=3": "[]",
		},
		statuses: map[string]int{},
	}
}

func TestCollectedFillsPastPullRequests(t *testing.T) {
	fx := prHeavyFixture()
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)

	issues, hasMore, err := client.ListIssues(context.Background(), "token", "o", "r", "open", 1, 7, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := issueNumbers(issues); !equalInts(got, []int{1, 2, 3, 4, 5, 6, 7}) {
		t.Fatalf("expected issues 1-7 in order, got %v", got)
	}
	if !hasMore {
		t.Fatalf("expected hasMore=true via lookahead")
	}
	if fx.calls != 2 {
		t.Fatalf("expected 2 upstream requests, got %d (%v)", fx.calls, fx.queries)
	}
}

func TestCollectedAdvancesAndDedupesAcrossPages(t *testing.T) {
	fx := prHeavyFixture()
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)
	ctx := context.Background()

	p1, _, err := client.ListIssues(ctx, "token", "o", "r", "open", 1, 7, "")
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	p2, hasMore, err := client.ListIssues(ctx, "token", "o", "r", "open", 2, 7, "")
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if got := issueNumbers(p2); !equalInts(got, []int{8, 9, 10, 11, 12}) {
		t.Fatalf("expected issues 8-12, got %v", got)
	}
	if hasMore {
		t.Fatalf("expected hasMore=false on exhausted results")
	}
	seen := map[int]bool{}
	for _, n := range append(issueNumbers(p1), issueNumbers(p2)...) {
		if seen[n] {
			t.Fatalf("duplicate issue %d across pages", n)
		}
		seen[n] = true
	}
	if len(seen) != 12 {
		t.Fatalf("expected 12 distinct issues, got %d", len(seen))
	}
}

func TestCollectedEmptyPage(t *testing.T) {
	fx := prHeavyFixture()
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)

	issues, hasMore, err := client.ListIssues(context.Background(), "token", "o", "r", "open", 5, 7, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(issues) != 0 || hasMore {
		t.Fatalf("expected empty page without more, got %+v hasMore=%v", issues, hasMore)
	}
}

func TestCollectedStatePropagated(t *testing.T) {
	for _, state := range []string{"open", "closed", "all"} {
		fx := &scriptedGitHub{
			pages:    map[string]string{"/repos/o/r/issues|page=1": "[]"},
			statuses: map[string]int{},
		}
		server := fx.serve(t)
		client := NewClient(server.URL)
		if _, _, err := client.ListIssues(context.Background(), "token", "o", "r", state, 1, 7, ""); err != nil {
			server.Close()
			t.Fatalf("state %q: %v", state, err)
		}
		server.Close()
		qs := queriesFor(fx, "/repos/o/r/issues")
		if len(qs) != 1 || !strings.Contains(qs[0], "state="+state) {
			t.Fatalf("state %q not propagated in %v", state, qs)
		}
	}
}

func TestCollectedSortsByUpdated(t *testing.T) {
	fx := &scriptedGitHub{
		pages:    map[string]string{"/repos/o/r/issues|page=1": "[]"},
		statuses: map[string]int{},
	}
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)
	if _, _, err := client.ListIssues(context.Background(), "token", "o", "r", "open", 1, 7, ""); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	qs := queriesFor(fx, "/repos/o/r/issues")
	if len(qs) != 1 || !strings.Contains(qs[0], "sort=updated") || !strings.Contains(qs[0], "direction=desc") {
		t.Fatalf("expected sort=updated&direction=desc in %v", qs)
	}
}

func TestCollectedCapBoundsUpstreamRequests(t *testing.T) {
	prs := make([]string, 0, 100)
	for n := 1; n <= 100; n++ {
		prs = append(prs, rawPR(1000+n))
	}
	page := "[" + strings.Join(prs, ",") + "]"
	pages := map[string]string{}
	for p := 1; p <= maxUpstreamPages; p++ {
		pages[fmt.Sprintf("/repos/o/r/issues|page=%d", p)] = page
	}
	fx := &scriptedGitHub{pages: pages, statuses: map[string]int{}}
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)

	issues, hasMore, err := client.ListIssues(context.Background(), "token", "o", "r", "closed", 1, 7, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no real issues, got %d", len(issues))
	}
	if !hasMore {
		t.Fatalf("expected hasMore=true when the safety cap stops a walk over full pages")
	}
	if fx.calls != maxUpstreamPages {
		t.Fatalf("expected exactly %d upstream requests, got %d", maxUpstreamPages, fx.calls)
	}
}

func TestCollectedUpstreamErrorsAreNotEmpty(t *testing.T) {
	fx := &scriptedGitHub{
		pages:    map[string]string{},
		statuses: map[string]int{"/repos/o/r/issues|page=1": http.StatusInternalServerError},
	}
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)
	if _, _, err := client.ListIssues(context.Background(), "token", "o", "r", "open", 1, 7, ""); !errors.Is(err, ErrGitHubServer) {
		t.Fatalf("expected ErrGitHubServer, got %v", err)
	}

	fx2 := &scriptedGitHub{
		pages:    map[string]string{},
		statuses: map[string]int{"/search/issues|page=1": http.StatusTooManyRequests},
	}
	server2 := fx2.serve(t)
	defer server2.Close()
	client2 := NewClient(server2.URL)
	if _, _, err := client2.ListIssues(context.Background(), "token", "o", "r", "open", 1, 7, "crash"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func searchFixture(itemsByPage map[string]string) *scriptedGitHub {
	pages := map[string]string{}
	for page, items := range itemsByPage {
		pages["/search/issues|page="+page] = searchEnvelope(items)
	}
	return &scriptedGitHub{pages: pages, statuses: map[string]int{}}
}

func decodedQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	u, err := url.ParseQuery(strings.TrimPrefix(raw, "/search/issues?"))
	if err != nil {
		t.Fatalf("bad query %q: %v", raw, err)
	}
	return u
}

func TestSearchTextMatchAcrossRepo(t *testing.T) {
	fx := searchFixture(map[string]string{
		"1": rawIssue(50, "open", false, "2024-03-01T00:00:00Z", `{"name":"bug"}`) + "," +
			rawIssue(51, "open", false, "2024-03-02T00:00:00Z", ""),
	})
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)

	issues, hasMore, err := client.ListIssues(context.Background(), "token", "o", "r", "open", 1, 7, "crash on save")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := issueNumbers(issues); !equalInts(got, []int{50, 51}) {
		t.Fatalf("expected issues 50,51, got %v", got)
	}
	if hasMore {
		t.Fatalf("expected hasMore=false on short search page")
	}
	if len(issues[0].Labels) != 1 || issues[0].Labels[0] != "bug" {
		t.Fatalf("expected label names preserved, got %+v", issues[0])
	}
	qs := queriesFor(fx, "/search/issues")
	if len(qs) == 0 {
		t.Fatalf("expected search endpoint to be hit, got %v", fx.queries)
	}
	q := decodedQuery(t, qs[0]).Get("q")
	for _, want := range []string{"crash on save", "repo:o/r", "type:issue", "state:open"} {
		if !strings.Contains(q, want) {
			t.Fatalf("expected %q in search q=%q", want, q)
		}
	}
}

func TestSearchNumericFindsIssueByNumber(t *testing.T) {
	fx := searchFixture(map[string]string{
		"1": rawIssue(190, "open", false, "2024-03-01T00:00:00Z", "") + "," +
			rawIssue(191, "open", false, "2024-03-02T00:00:00Z", ""),
	})
	fx.pages["/repos/o/r/issues/172|page="] = `{"number":172,"title":"Unrelated title","body":"no digits here","state":"open","labels":[],"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-03-03T00:00:00Z","html_url":"https://github.com/o/r/issues/172","user":{"login":"carol"}}`
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)

	issues, _, err := client.ListIssues(context.Background(), "token", "o", "r", "open", 1, 7, "172")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := issueNumbers(issues); !equalInts(got, []int{172, 190, 191}) {
		t.Fatalf("expected direct hit 172 first, got %v", got)
	}
}

func TestSearchNumericRespectsState(t *testing.T) {
	fx := searchFixture(map[string]string{
		"1": rawIssue(190, "open", false, "2024-03-01T00:00:00Z", ""),
	})
	fx.pages["/repos/o/r/issues/172|page="] = `{"number":172,"title":"Old bug","body":"x","state":"closed","labels":[],"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z","html_url":"https://github.com/o/r/issues/172","user":{"login":"carol"}}`
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)

	issues, _, err := client.ListIssues(context.Background(), "token", "o", "r", "open", 1, 7, "172")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := issueNumbers(issues); !equalInts(got, []int{190}) {
		t.Fatalf("expected state-mismatched direct hit excluded, got %v", got)
	}
}

func TestSearchExcludesPullRequests(t *testing.T) {
	fx := searchFixture(map[string]string{
		"1": rawIssue(60, "open", false, "2024-03-01T00:00:00Z", "") + "," +
			rawIssue(61, "open", true, "2024-03-02T00:00:00Z", ""),
	})
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)

	issues, _, err := client.ListIssues(context.Background(), "token", "o", "r", "open", 1, 7, "thing")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := issueNumbers(issues); !equalInts(got, []int{60}) {
		t.Fatalf("expected PR item dropped, got %v", got)
	}
	qs := queriesFor(fx, "/search/issues")
	if len(qs) == 0 || !strings.Contains(decodedQuery(t, qs[0]).Get("q"), "type:issue") {
		t.Fatalf("expected type:issue qualifier in %v", qs)
	}
}

func TestSearchScopeAndStateQualifiers(t *testing.T) {
	fx := searchFixture(map[string]string{"1": ""})
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)

	if _, _, err := client.ListIssues(context.Background(), "token", "o", "r", "closed", 1, 7, "timeout"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	qs := queriesFor(fx, "/search/issues")
	if len(qs) == 0 {
		t.Fatalf("expected search endpoint hit, got %v", fx.queries)
	}
	q := decodedQuery(t, qs[0]).Get("q")
	for _, want := range []string{"repo:o/r", "type:issue", "state:closed"} {
		if !strings.Contains(q, want) {
			t.Fatalf("expected %q in search q=%q", want, q)
		}
	}
	if strings.Contains(q, "state:open") || strings.Contains(q, "state:all") {
		t.Fatalf("unexpected state qualifier in q=%q", q)
	}
}

func TestBlankQueryUsesRecentListing(t *testing.T) {
	fx := &scriptedGitHub{
		pages:    map[string]string{"/repos/o/r/issues|page=1": "[]"},
		statuses: map[string]int{},
	}
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)

	if _, _, err := client.ListIssues(context.Background(), "token", "o", "r", "open", 1, 7, "   "); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(queriesFor(fx, "/search/issues")) != 0 {
		t.Fatalf("blank query must not hit the search endpoint: %v", fx.queries)
	}
	if len(queriesFor(fx, "/repos/o/r/issues")) != 1 {
		t.Fatalf("blank query must use the recent listing: %v", fx.queries)
	}
}

func TestSearchPaginationAdvances(t *testing.T) {
	items := make([]string, 0, 8)
	for n := 1; n <= 8; n++ {
		items = append(items, rawIssue(n, "open", false, "2024-03-01T00:00:00Z", ""))
	}
	fx := searchFixture(map[string]string{"1": strings.Join(items, ",")})
	server := fx.serve(t)
	defer server.Close()
	client := NewClient(server.URL)
	ctx := context.Background()

	p1, hasMore, err := client.ListIssues(ctx, "token", "o", "r", "open", 1, 7, "thing")
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(p1) != 7 || !hasMore {
		t.Fatalf("expected full page with more, got %d hasMore=%v", len(p1), hasMore)
	}
	// No second search page exists in the fixture (page 2 → empty).
	p2, hasMore2, err := client.ListIssues(ctx, "token", "o", "r", "open", 2, 7, "thing")
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(p2) != 1 || hasMore2 {
		t.Fatalf("expected final short page without more, got %d hasMore=%v", len(p2), hasMore2)
	}
	seen := map[int]bool{}
	for _, n := range append(issueNumbers(p1), issueNumbers(p2)...) {
		if seen[n] {
			t.Fatalf("duplicate issue %d across search pages", n)
		}
		seen[n] = true
	}
}
