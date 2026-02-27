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

func TestWeatherToolExecute_Single(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "j1", r.URL.Query().Get("format"))
		assert.NotEmpty(t, strings.TrimPrefix(r.URL.Path, "/"))
		_, _ = io.WriteString(w, weatherFixtureJSON())
	}))
	defer server.Close()

	tool := &WeatherTool{
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"location":"San Francisco, CA"}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))

	assert.Equal(t, "San Francisco, CA", resp["query_location"])
	assert.Equal(t, "San Francisco, California, United States", resp["location"])
	assert.Equal(t, "2026-02-27", resp["start"])

	forecast, ok := resp["forecast"].([]interface{})
	require.True(t, ok)
	assert.Len(t, forecast, 3)
}

func TestWeatherToolExecute_BatchAndStartDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, weatherFixtureJSON())
	}))
	defer server.Close()

	tool := &WeatherTool{
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"weather":[{"location":"Bandung"},{"location":"Tokyo","start":"2026-02-28","duration":1}]}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))

	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 2)

	second, ok := results[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "2026-02-28", second["start"])

	secondForecast, ok := second["forecast"].([]interface{})
	require.True(t, ok)
	require.Len(t, secondForecast, 1)

	firstDay, ok := secondForecast[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "2026-02-28", firstDay["date"])
}

func TestWeatherToolExecute_InvalidStartDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, weatherFixtureJSON())
	}))
	defer server.Close()

	tool := &WeatherTool{
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"location":"Tokyo","start":"2026/02/28"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "YYYY-MM-DD")
}

func TestWeatherToolExecute_RequiresLocation(t *testing.T) {
	tool := &WeatherTool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "location is required")
}

func weatherFixtureJSON() string {
	return `{
  "current_condition": [
    {
      "temp_C": "12",
      "FeelsLikeC": "10",
      "weatherDesc": [{"value":"Partly cloudy"}],
      "humidity": "81",
      "windspeedKmph": "15",
      "observation_time": "04:30 PM"
    }
  ],
  "nearest_area": [
    {
      "areaName": [{"value":"San Francisco"}],
      "region": [{"value":"California"}],
      "country": [{"value":"United States"}]
    }
  ],
  "weather": [
    {
      "date": "2026-02-27",
      "maxtempC": "14",
      "mintempC": "9",
      "avgtempC": "11",
      "hourly": [{"weatherDesc": [{"value":"Cloudy"}]}]
    },
    {
      "date": "2026-02-28",
      "maxtempC": "15",
      "mintempC": "10",
      "avgtempC": "12",
      "hourly": [{"weatherDesc": [{"value":"Sunny"}]}]
    },
    {
      "date": "2026-03-01",
      "maxtempC": "16",
      "mintempC": "11",
      "avgtempC": "13",
      "hourly": [{"weatherDesc": [{"value":"Clear"}]}]
    }
  ]
}`
}
