package stathat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAlerts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Alert{
			{ID: 1, StatID: "s1", StatName: "hits", Kind: AlertKindValue, TimeWindow: "1h"},
			{ID: 2, StatID: "s2", StatName: "errors", Kind: AlertKindData, TimeWindow: "5m"},
		})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	alerts, err := c.ListAlerts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 2 {
		t.Fatalf("got %d alerts, want 2", len(alerts))
	}
	if alerts[0].Kind != AlertKindValue {
		t.Errorf("first alert kind: got %q, want %q", alerts[0].Kind, AlertKindValue)
	}
}

func TestGetAlert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/x/tok/alerts/999" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"msg":"not found"}`))
			return
		}
		json.NewEncoder(w).Encode(Alert{ID: 1, StatName: "hits", Kind: AlertKindValue})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))

	t.Run("found", func(t *testing.T) {
		alert, err := c.GetAlert(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if alert.StatName != "hits" {
			t.Errorf("stat_name: got %q, want %q", alert.StatName, "hits")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := c.GetAlert(context.Background(), 999)
		if !errors.Is(err, ErrAlertNotFound) {
			t.Errorf("expected ErrAlertNotFound, got %v", err)
		}
	})
}

func TestCreateValueAlert(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		json.NewEncoder(w).Encode(Alert{
			ID: 10, StatID: "s1", Kind: AlertKindValue,
			TimeWindow: "1h", Operator: "greater than", Threshold: 100,
		})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	alert, err := c.CreateAlert(context.Background(), CreateAlertParams{
		StatID:     "s1",
		Kind:       AlertKindValue,
		TimeWindow: TimeWindow1h,
		Operator:   "greater than",
		Threshold:  ptrFloat64(100),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert.ID != 10 {
		t.Errorf("alert ID: got %d, want 10", alert.ID)
	}

	vals, _ := parseFormBody(gotBody)
	assertFormValue(t, vals, "stat_id", "s1")
	assertFormValue(t, vals, "kind", "value")
	assertFormValue(t, vals, "time_window", "1h")
	assertFormValue(t, vals, "operator", "greater than")
	assertFormValue(t, vals, "threshold", "100")
}

func TestCreateDeltaAlert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		vals, _ := parseFormBody(string(body))
		if vals["kind"] != "delta" {
			t.Errorf("kind: got %q, want delta", vals["kind"])
		}
		if vals["percentage"] != "50" {
			t.Errorf("percentage: got %q, want 50", vals["percentage"])
		}
		if vals["time_delta"] != "1d" {
			t.Errorf("time_delta: got %q, want 1d", vals["time_delta"])
		}
		json.NewEncoder(w).Encode(Alert{ID: 11, Kind: AlertKindDelta})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	alert, err := c.CreateAlert(context.Background(), CreateAlertParams{
		StatID:     "s2",
		Kind:       AlertKindDelta,
		TimeWindow: TimeWindow1d,
		Operator:   "different than",
		Percentage: ptrFloat64(50),
		TimeDelta:  "1d",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert.Kind != AlertKindDelta {
		t.Errorf("kind: got %q, want %q", alert.Kind, AlertKindDelta)
	}
}

func TestCreateDataAlert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		vals, _ := parseFormBody(string(body))
		if vals["kind"] != "data" {
			t.Errorf("kind: got %q, want data", vals["kind"])
		}
		// Data alerts should not have threshold/percentage params
		if _, ok := vals["threshold"]; ok {
			t.Error("data alert should not have threshold")
		}
		json.NewEncoder(w).Encode(Alert{ID: 12, Kind: AlertKindData})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	_, err := c.CreateAlert(context.Background(), CreateAlertParams{
		StatID:     "s3",
		Kind:       AlertKindData,
		TimeWindow: TimeWindow5m,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAlert(t *testing.T) {
	var gotPath string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		json.NewEncoder(w).Encode(deleteResponse{Msg: "alert deleted."})
	}))
	defer srv.Close()

	c := New(WithAccessToken("tok"), WithExportURL(srv.URL))
	err := c.DeleteAlert(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method: got %s, want DELETE", gotMethod)
	}
	if want := "/x/tok/alerts/42"; gotPath != want {
		t.Errorf("path: got %q, want %q", gotPath, want)
	}
}

func ptrFloat64(v float64) *float64 { return &v }
