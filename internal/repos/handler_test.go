package repos

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func testPaginationContext(t *testing.T, rawQuery string) (int, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/repos/x/issues?"+rawQuery, nil)
	return parsePaginationWithDefault(c, IssueDefaultPerPage)
}

func TestParsePaginationIssuesDefault(t *testing.T) {
	page, perPage := testPaginationContext(t, "")
	if page != 1 || perPage != IssueDefaultPerPage {
		t.Fatalf("expected 1/%d, got %d/%d", IssueDefaultPerPage, page, perPage)
	}
}

func TestParsePaginationInvalidParams(t *testing.T) {
	page, perPage := testPaginationContext(t, "page=0&per_page=0")
	if page != 1 || perPage != IssueDefaultPerPage {
		t.Fatalf("expected 1/%d, got %d/%d", IssueDefaultPerPage, page, perPage)
	}
	page, perPage = testPaginationContext(t, "page=abc&per_page=500")
	if page != 1 || perPage != 100 {
		t.Fatalf("expected 1/100, got %d/%d", page, perPage)
	}
	page, perPage = testPaginationContext(t, "page=3&per_page=7")
	if page != 3 || perPage != 7 {
		t.Fatalf("expected 3/7, got %d/%d", page, perPage)
	}
}
