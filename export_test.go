package stathat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %s, want GET", r.Method)
		}
		json.NewEncoder(w).Encode([]Stat{
			{ID: "abc", Name: "hits", Public: false, Counter: true},
			{ID: "def", Name: "latency", Public: true, Counter: false},
		})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok123"), WithExportURL(srv.URL))
	stats, err := c.StatList(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("got %d stats, want 2", len(stats))
	}
	if stats[0].Name != "hits" {
		t.Errorf("first stat name: got %q, want %q", stats[0].Name, "hits")
	}
}

func TestStatListPagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		offset := r.URL.Query().Get("offset")

		var stats []Stat
		if offset == "" || offset == "0" {
			// First page: return exactly pageSize items
			for i := 0; i < pageSize; i++ {
				stats = append(stats, Stat{ID: fmt.Sprintf("s%d", i), Name: fmt.Sprintf("stat%d", i)})
			}
		} else {
			// Second page: return fewer than pageSize
			for i := 0; i < 5; i++ {
				stats = append(stats, Stat{ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("page2-%d", i)})
			}
		}
		json.NewEncoder(w).Encode(stats)
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	stats, err := c.StatList(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats) != pageSize+5 {
		t.Errorf("got %d stats, want %d", len(stats), pageSize+5)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
}

func TestStatIter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Stat{
			{ID: "a", Name: "alpha"},
			{ID: "b", Name: "beta"},
		})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	var names []string
	for s, err := range c.StatIter(context.Background()) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		names = append(names, s.Name)
	}
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("got %v, want [alpha beta]", names)
	}
}

func TestStatInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "missing" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"msg":"not found"}`))
			return
		}
		json.NewEncoder(w).Encode(Stat{ID: "abc", Name: name, Counter: true})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))

	t.Run("found", func(t *testing.T) {
		stat, err := c.StatInfo(context.Background(), "page views")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stat.Name != "page views" {
			t.Errorf("name: got %q, want %q", stat.Name, "page views")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := c.StatInfo(context.Background(), "missing")
		if !errors.Is(err, ErrStatNotFound) {
			t.Errorf("expected ErrStatNotFound, got %v", err)
		}
	})
}

func TestDeleteStat(t *testing.T) {
	var gotPath string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		json.NewEncoder(w).Encode(deleteResponse{Msg: "stat deleted."})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	err := c.DeleteStat(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method: got %s, want DELETE", gotMethod)
	}
	if want := "/x/tok/stats/abc123"; gotPath != want {
		t.Errorf("path: got %q, want %q", gotPath, want)
	}
}

func TestGetData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request path and query
		if r.URL.Query().Get("t") != "1w3h" {
			t.Errorf("timeframe param: got %q, want %q", r.URL.Query().Get("t"), "1w3h")
		}
		json.NewEncoder(w).Encode([]Dataset{
			{
				Name:      "hits",
				Timeframe: "1w3h",
				Points: []DataPoint{
					{Time: 1700000000, Value: 42},
					{Time: 1700010800, Value: 55},
				},
			},
		})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	datasets, err := c.GetData(context.Background(), DataQuery{
		StatIDs:   []string{"stat1"},
		Timeframe: NewTimeframe(1, Week, 3, Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(datasets) != 1 {
		t.Fatalf("got %d datasets, want 1", len(datasets))
	}
	if len(datasets[0].Points) != 2 {
		t.Errorf("got %d points, want 2", len(datasets[0].Points))
	}
}

func TestGetDataMultiStat(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode([]Dataset{
			{Name: "stat1"},
			{Name: "stat2"},
		})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	_, err := c.GetData(context.Background(), DataQuery{
		StatIDs:   []string{"id1", "id2"},
		Timeframe: RawTimeframe("1d1h"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "/x/tok/data/id1/id2"; gotPath != want {
		t.Errorf("path: got %q, want %q", gotPath, want)
	}
}

func TestGetStatData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Dataset{
			{Name: "cpu", Timeframe: "1d1h", Points: []DataPoint{{Time: 1700000000, Value: 0.5}}},
		})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	ds, err := c.GetStatData(context.Background(), "cpu_id", NewTimeframe(1, Day, 1, Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ds.Name != "cpu" {
		t.Errorf("name: got %q, want %q", ds.Name, "cpu")
	}
}

func TestNoAccessToken(t *testing.T) {
	c := New()
	_, err := c.StatList(context.Background())
	if !errors.Is(err, ErrNoAccessToken) {
		t.Errorf("expected ErrNoAccessToken, got %v", err)
	}
}
