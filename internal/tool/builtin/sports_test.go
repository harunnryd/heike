package builtin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSportsToolExecute_Schedule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasSuffix(r.URL.Path, "/basketball/nba/scoreboard"))
		assert.Equal(t, "LAL", r.URL.Query().Get("team"))
		assert.Equal(t, "20260227-20260228", r.URL.Query().Get("dates"))
		_, _ = io.WriteString(w, `{
			"events":[
				{"id":"1","name":"Lakers vs Warriors","date":"2026-02-27T20:00Z","status":{"type":{"description":"Scheduled"}},"competitions":[{"competitors":[{"team":{"shortDisplayName":"Lakers","abbreviation":"LAL"},"score":"0"},{"team":{"shortDisplayName":"Warriors","abbreviation":"GSW"},"score":"0"}]}]},
				{"id":"2","name":"Celtics vs Knicks","date":"2026-02-27T22:00Z","status":{"type":{"description":"Scheduled"}},"competitions":[{"competitors":[{"team":{"shortDisplayName":"Celtics","abbreviation":"BOS"},"score":"0"},{"team":{"shortDisplayName":"Knicks","abbreviation":"NYK"},"score":"0"}]}]}
			]
		}`)
	}))
	defer server.Close()

	tool := &SportsTool{
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"fn":"schedule","league":"nba","team":"LAL","date_from":"2026-02-27","date_to":"2026-02-28","num_games":5}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Equal(t, "schedule", resp["fn"])

	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 1)
	first, ok := results[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Lakers", first["home_team"])
	assert.Equal(t, "Warriors", first["away_team"])
}

func TestSportsToolExecute_Standings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasSuffix(r.URL.Path, "/football/nfl/standings"))
		_, _ = io.WriteString(w, `{
			"children":[
				{"standings":{"entries":[
					{"team":{"displayName":"Chiefs"},"stats":[{"name":"rank","displayValue":"1"},{"name":"record","displayValue":"12-5"},{"name":"points","displayValue":"0"}]},
					{"team":{"displayName":"Bills"},"stats":[{"name":"rank","displayValue":"2"},{"name":"record","displayValue":"11-6"},{"name":"points","displayValue":"0"}]}
				]}}
			]
		}`)
	}))
	defer server.Close()

	tool := &SportsTool{
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"fn":"standings","league":"nfl"}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))

	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 2)
	first, ok := results[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Chiefs", first["team"])
	assert.Equal(t, "1", first["rank"])
}

func TestSportsToolExecute_UnsupportedLeague(t *testing.T) {
	tool := &SportsTool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"fn":"schedule","league":"liga1"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported league")
}
