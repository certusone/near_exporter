package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (c *nearExporter) getValidatorInfo() (response *ValidatorsResponse, err error) {
	req, err := http.NewRequest("POST", c.rpcAddr,
		bytes.NewBufferString(`{"jsonrpc":"2.0","id":1, "method":"validators", "params":[null]}`))
	if err != nil {
		panic(err)
	}

	req.Header.Set("content-type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error during validators request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("return status code %d, response: %s", resp.StatusCode, body)

	}

	if err = json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response body: %w", err)
	}

	if response.Error.Code != 0 {
		return nil, fmt.Errorf("JSONRPC error: %s", body)
	}

	return
}
