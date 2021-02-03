package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type NEARStatusResponse struct {
	Version struct {
		Version string `json:"version"`
		Build   string `json:"build"`
	} `json:"version"`
	ChainID               string `json:"chain_id"`
	ProtocolVersion       int    `json:"protocol_version"`
	LatestProtocolVersion int    `json:"latest_protocol_version"`
	RPCAddr               string `json:"rpc_addr"`
	Validators            []struct {
		AccountID string `json:"account_id"`
		IsSlashed bool   `json:"is_slashed"`
	} `json:"validators"`
	SyncInfo struct {
		LatestBlockHash   string    `json:"latest_block_hash"`
		LatestBlockHeight int       `json:"latest_block_height"`
		LatestStateRoot   string    `json:"latest_state_root"`
		LatestBlockTime   time.Time `json:"latest_block_time"`
		Syncing           bool      `json:"syncing"`
	} `json:"sync_info"`
	ValidatorAccountID string `json:"validator_account_id"`
}

// checkSyncStatus determines whether the NEAR node is currently syncing the chain.
func (c *nearExporter) checkSyncStatus() (bool, error) {
	r, err := http.Get(c.rpcAddr + "/status")
	if err != nil {
		return false, fmt.Errorf("failed to request /status: %w", err)
	}

	defer r.Body.Close()
	r.Header.Set("content-type", "application/json")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	if r.StatusCode != 200 {
		return false, fmt.Errorf("status code %d, response: %s", r.StatusCode, body)
	}

	var res NEARStatusResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return false, fmt.Errorf("error decoding: %w, response: %s", err, body)
	}

	return res.SyncInfo.Syncing, nil
}
