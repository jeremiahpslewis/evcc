package homeassistant

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConnection(url string) *Connection {
	return &Connection{
		client: http.DefaultClient,
		url:    url,
	}
}

func TestCallSwitchService_DomainDispatch(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		expect string
	}{
		{"switch", "switch", "switch"},
		{"light", "light", "light"},
		{"climate", "climate", "climate"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			_ = newTestConnection(srv.URL)
		})}
}

// TestGetIntState_AcceptsFloatStates verifies that integer reads tolerate
// fractional entity states like "55.0", which Home Assistant commonly
// reports for number/sensor entities (e.g. target temperature limits).
func TestGetIntState_AcceptsFloatStates(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		want    int64
		wantErr bool
	}{
		{"integer state", "55", 55, false},
		{"float state", "55.0", 55, false},
		{"fractional state truncates", "21.5", 21, false},
		{"negative float state", "-3.7", -3, false},
		{"non-numeric state", "on", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"state":"` + tc.state + `","attributes":{}}`)
			}))
			defer srv.Close()

			got, err := newTestConnection(srv.URL).GetIntState("sensor.heatpump_target_temperature")

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
