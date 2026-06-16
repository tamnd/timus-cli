package timus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/timus-cli/timus"
)

const testHTML = `
<TR CLASS="content"><TD><BR></TD><TD>1000</TD><TD CLASS="name"><A HREF="problem.aspx?space=1&amp;num=1000">A+B Problem</A></TD><TD CLASS="source"></TD><TD><A HREF="rating.aspx?space=1&num=1000">107355</A></TD><TD>16</TD></TR>
<TR CLASS="content"><TD><BR></TD><TD>1001</TD><TD CLASS="name"><A HREF="problem.aspx?space=1&amp;num=1001">Reverse Root</A></TD><TD CLASS="source"></TD><TD><A HREF="rating.aspx?space=1&num=1001">18539</A></TD><TD>498</TD></TR>
<TR CLASS="content"><TD><BR></TD><TD>1002</TD><TD CLASS="name"><A HREF="problem.aspx?space=1&amp;num=1002">Phone Numbers</A></TD><TD CLASS="source"></TD><TD><A HREF="rating.aspx?space=1&num=1002">6543</A></TD><TD>723</TD></TR>
`

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "1" {
			_, _ = w.Write([]byte(testHTML))
		} else {
			// no page=2 link → hasMore is false
			_, _ = w.Write([]byte("<html></html>"))
		}
	}))
}

func newTestClient(srv *httptest.Server) *timus.Client {
	cfg := timus.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	return timus.NewClient(cfg)
}

func TestList(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := newTestClient(srv)
	probs, err := c.List(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(probs) != 3 {
		t.Fatalf("got %d problems, want 3", len(probs))
	}
	p := probs[0]
	if p.Num != 1000 {
		t.Errorf("Num = %d, want 1000", p.Num)
	}
	if p.Title != "A+B Problem" {
		t.Errorf("Title = %q, want %q", p.Title, "A+B Problem")
	}
	if p.Solved != 107355 {
		t.Errorf("Solved = %d, want 107355", p.Solved)
	}
	if p.Difficulty != 16 {
		t.Errorf("Difficulty = %d, want 16", p.Difficulty)
	}
}

func TestListLimit(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := newTestClient(srv)
	probs, err := c.List(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(probs) != 2 {
		t.Fatalf("got %d problems, want 2", len(probs))
	}
}

func TestSearch(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := newTestClient(srv)
	results, err := c.Search(context.Background(), "root", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Num != 1001 {
		t.Errorf("Num = %d, want 1001", results[0].Num)
	}
}
