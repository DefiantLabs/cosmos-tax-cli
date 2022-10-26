package osmosis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DefiantLabs/cosmos-tax-cli/config"
	"go.uber.org/zap"
	"io"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"time"

	tmjson "github.com/tendermint/tendermint/libs/json"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	jsonrpc "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	types "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

func DoHttpReq(url string, authHeader string) (*http.Response, error) {
	// Send req using http Client
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", authHeader)
	return client.Do(req)
}

func argsToURLValues(args map[string]interface{}) (url.Values, error) {
	values := make(url.Values)
	if len(args) == 0 {
		return values, nil
	}

	err := argsToJSON(args)
	if err != nil {
		return nil, err
	}

	for key, val := range args {
		values.Set(key, val.(string))
	}

	return values, nil
}

func argsToJSON(args map[string]interface{}) error {
	for k, v := range args {
		rt := reflect.TypeOf(v)
		isByteSlice := rt.Kind() == reflect.Slice && rt.Elem().Kind() == reflect.Uint8
		if isByteSlice {
			bytes := reflect.ValueOf(v).Bytes()
			args[k] = fmt.Sprintf("0x%X", bytes)
			continue
		}

		data, err := tmjson.Marshal(v)
		if err != nil {
			return err
		}
		args[k] = string(data)
	}
	return nil
}

// Call issues a POST form HTTP request.
func (c *URIClient) DoHttpGetWithRetry(ctx context.Context, method string, params map[string]interface{}, result interface{}, attmptNum int) (interface{}, error) {
	values, err := argsToURLValues(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode params: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Address+"/"+method, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating new request: %w", err)
	}

	req.URL.RawQuery = values.Encode()
	//fmt.Printf("Query string: %s\n", values.Encode())

	// req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.AuthHeader != "" {
		req.Header.Add("Authorization", c.AuthHeader)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	responseUnmarshalled, err := unmarshalResponseBytes(responseBytes, jsonrpc.URIClientRequestID, result)
	if err != nil {
		if len(responseBytes) < 2 && attmptNum < 5 {
			attmptNum++
			config.Log.Warn(fmt.Sprintf("Unmarshal failed... reattempt #%v.", attmptNum-1), zap.Error(err))
			time.Sleep(time.Second * time.Duration(math.Pow(2, float64(attmptNum))))
			return c.DoHttpGetWithRetry(ctx, method, params, result, attmptNum)
		}
		config.Log.Error("Unmarshal never succeeded... giving up.", zap.Error(err))
		return responseUnmarshalled, err
	}
	return responseUnmarshalled, err
}

// From the JSON-RPC 2.0 spec:
// id: It MUST be the same as the value of the id member in the Request Object.
func validateAndVerifyID(res *types.RPCResponse, expectedID types.JSONRPCIntID) error {
	if err := validateResponseID(res.ID); err != nil {
		return err
	}
	if expectedID != res.ID.(types.JSONRPCIntID) { // validateResponseID ensured res.ID has the right type
		return fmt.Errorf("response ID (%d) does not match request ID (%d)", res.ID, expectedID)
	}
	return nil
}

func validateResponseID(id interface{}) error {
	if id == nil {
		return errors.New("no ID")
	}
	_, ok := id.(types.JSONRPCIntID)
	if !ok {
		return fmt.Errorf("expected JSONRPCIntID, but got: %T", id)
	}
	return nil
}

func unmarshalResponseBytes(responseBytes []byte, expectedID types.JSONRPCIntID, result interface{}) (interface{}, error) {
	// Read response.  If rpc/core/types is imported, the result will unmarshal
	// into the correct type.
	response := &types.RPCResponse{}
	if err := json.Unmarshal(responseBytes, response); err != nil {
		return nil, fmt.Errorf("error unmarshalling: %w", err)
	}

	if response.Error != nil {
		return nil, response.Error
	}

	if err := validateAndVerifyID(response, expectedID); err != nil {
		return nil, fmt.Errorf("wrong ID: %w", err)
	}

	// Unmarshal the RawMessage into the result.
	if err := tmjson.Unmarshal(response.Result, result); err != nil {
		return nil, fmt.Errorf("error unmarshalling result: %w", err)
	}

	return result, nil
}
func (c *URIClient) DoBlockSearch(ctx context.Context, query string, page, perPage *int, orderBy string) (*ctypes.ResultBlockSearch, error) {
	result := new(ctypes.ResultBlockSearch)
	params := map[string]interface{}{
		"query":    query,
		"order_by": orderBy,
	}

	if page != nil {
		params["page"] = page
	}
	if perPage != nil {
		params["per_page"] = perPage
	}

	_, err := c.DoHttpGetWithRetry(ctx, "block_search", params, result, 0)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *URIClient) DoBlockResults(ctx context.Context, height *int64) (*ctypes.ResultBlockResults, error) {
	result := new(ctypes.ResultBlockResults)
	params := make(map[string]interface{})
	if height != nil {
		params["height"] = height
	}

	_, err := c.DoHttpGetWithRetry(ctx, "block_results", params, result, 0)
	if err != nil {
		return nil, err
	}

	return result, nil
}
