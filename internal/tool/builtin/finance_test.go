package builtin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinanceToolExecute_Single(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "AMD", r.URL.Query().Get("symbols"))
		_, _ = io.WriteString(w, `{"quoteResponse":{"result":[{"symbol":"AMD","regularMarketPrice":190.5,"regularMarketChange":1.2,"regularMarketChangePercent":0.63,"currency":"USD","regularMarketTime":1767139200,"marketState":"REGULAR"}]}}`)
	}))
	defer server.Close()

	tool := &FinanceTool{
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"ticker":"AMD","type":"equity","market":"USA"}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))

	assert.Equal(t, true, resp["found"])
	assert.Equal(t, "AMD", resp["symbol"])
	assert.Equal(t, "equity", resp["type"])
	assert.Equal(t, "AMD", resp["resolved_symbol"])
}

func TestFinanceToolExecute_Batch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "BTC-USD,SPY", r.URL.Query().Get("symbols"))
		_, _ = io.WriteString(w, `{"quoteResponse":{"result":[{"symbol":"BTC-USD","regularMarketPrice":60123.4,"currency":"USD"},{"symbol":"SPY","regularMarketPrice":540.2,"currency":"USD"}]}}`)
	}))
	defer server.Close()

	tool := &FinanceTool{
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"finance":[{"ticker":"BTC","type":"crypto","market":""},{"ticker":"SPY","type":"fund","market":"USA"}]}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))

	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 2)

	first, ok := results[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "BTC-USD", first["resolved_symbol"])
	assert.Equal(t, true, first["found"])
}

func TestFinanceToolExecute_UnsupportedType(t *testing.T) {
	tool := &FinanceTool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"ticker":"ABC","type":"bond"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported finance type")
}
