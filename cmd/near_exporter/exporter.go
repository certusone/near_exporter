package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type ValidatorsResponse struct {
	ID      int    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  Result `json:"result"`
	Error   Error  `json:"error"`
}

type Error struct {
	Code    int    `json:"code"`
	Data    string `json:"data"`
	Message string `json:"message"`
}

type CurrentProposals struct {
	AccountID string `json:"account_id"`
	PublicKey string `json:"public_key"`
	Stake     string `json:"stake"`
}

type CurrentValidator struct {
	AccountID         string `json:"account_id"`
	IsSlashed         bool   `json:"is_slashed"`
	NumExpectedBlocks int    `json:"num_expected_blocks"`
	NumProducedBlocks int    `json:"num_produced_blocks"`
	PublicKey         string `json:"public_key"`
	Shards            []int  `json:"shards"`
	Stake             string `json:"stake"`
}

type NextValidator struct {
	AccountID string `json:"account_id"`
	PublicKey string `json:"public_key"`
	Shards    []int  `json:"shards"`
	Stake     string `json:"stake"`
}

type Result struct {
	CurrentFishermen  []interface{}      `json:"current_fishermen"`
	CurrentProposals  []CurrentProposals `json:"current_proposals"`
	CurrentValidators []CurrentValidator `json:"current_validators"`
	EpochStartHeight  int                `json:"epoch_start_height"`
	NextFishermen     []interface{}      `json:"next_fishermen"`
	NextValidators    []NextValidator    `json:"next_validators"`
	PrevEpochKickout  []interface{}      `json:"prev_epoch_kickout"`
}

const (
	httpTimeout = 2 * time.Second
)

var (
	listenAddr  = os.Getenv("LISTEN_ADDR")
	nearRPCAddr = os.Getenv("NEAR_RPC_ADDR")
)

func init() {
	if nearRPCAddr == "" {
		log.Fatal("Please specify NEAR_RPC_ADDR")
	}

	if listenAddr == "" {
		listenAddr = ":8080"
	}
}

type nearExporter struct {
	client  *http.Client
	rpcAddr string

	totalValidatorsDesc     *prometheus.Desc
	epochStartHeight        *prometheus.Desc
	validatorStake          *prometheus.Desc
	validatorExpectedBlocks *prometheus.Desc
	validatorProducedBlocks *prometheus.Desc
	validatorIsSlashed      *prometheus.Desc
}

func NewSolanaCollector(rpcAddr string) prometheus.Collector {
	return &nearExporter{
		client:  &http.Client{Timeout: httpTimeout},
		rpcAddr: rpcAddr,
		totalValidatorsDesc: prometheus.NewDesc(
			"near_active_validators",
			"Total number of active validators",
			nil, nil),
		epochStartHeight: prometheus.NewDesc(
			"near_epoch_start_height",
			"Current epoch's start height",
			nil, nil),
		validatorStake: prometheus.NewDesc(
			"near_validator_stake",
			"Validator's stake",
			[]string{"account_id"}, nil),
		validatorExpectedBlocks: prometheus.NewDesc(
			"near_validator_expected_blocks",
			"Validators's expected blocks",
			[]string{"account_id"}, nil),
		validatorProducedBlocks: prometheus.NewDesc(
			"near_validator_produced_blocks",
			"Validator's actual produced blocks",
			[]string{"account_id"}, nil),
		validatorIsSlashed: prometheus.NewDesc(
			"near_validator_is_slashed",
			"Whether the validator is slashed",
			[]string{"account_id"}, nil),
	}
}

func (collector nearExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.totalValidatorsDesc
}

func (collector nearExporter) mustEmitMetrics(ch chan<- prometheus.Metric, response *ValidatorsResponse) {
	ch <- prometheus.MustNewConstMetric(collector.totalValidatorsDesc, prometheus.GaugeValue,
		float64(len(response.Result.CurrentValidators)))
	ch <- prometheus.MustNewConstMetric(collector.epochStartHeight, prometheus.GaugeValue,
		float64(response.Result.EpochStartHeight))

	for _, validator := range response.Result.CurrentValidators {
		stake, err := strconv.ParseFloat(validator.Stake, 64)
		if err != nil {
			ch <- prometheus.NewInvalidMetric(collector.validatorStake, fmt.Errorf("invalid stake: %s", validator.Stake))
		} else {
			ch <- prometheus.MustNewConstMetric(collector.validatorStake, prometheus.GaugeValue,
				stake, validator.AccountID)
		}

		ch <- prometheus.MustNewConstMetric(collector.validatorExpectedBlocks, prometheus.GaugeValue,
			float64(validator.NumExpectedBlocks), validator.AccountID)
		ch <- prometheus.MustNewConstMetric(collector.validatorProducedBlocks, prometheus.GaugeValue,
			float64(validator.NumProducedBlocks), validator.AccountID)

		if validator.IsSlashed {
			ch <- prometheus.MustNewConstMetric(collector.validatorIsSlashed, prometheus.GaugeValue, 1, validator.AccountID)
		} else {
			ch <- prometheus.MustNewConstMetric(collector.validatorIsSlashed, prometheus.GaugeValue, 0, validator.AccountID)
		}
	}
}

func (collector nearExporter) Collect(ch chan<- prometheus.Metric) {
	var (
		validatorResponse ValidatorsResponse
		body              []byte
		err               error
	)

	req, err := http.NewRequest("POST", collector.rpcAddr,
		bytes.NewBufferString(`{"jsonrpc":"2.0","id":1, "method":"validators", "params":[null]}`))
	if err != nil {
		panic(err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := collector.client.Do(req)
	if err != nil {
		goto error
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		goto error
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("status code %d, response: %s", resp.StatusCode, body)
		goto error
	}

	if err = json.Unmarshal(body, &validatorResponse); err != nil {
		goto error
	}

	if validatorResponse.Error.Code != 0 {
		err = fmt.Errorf("JSONRPC error: %s", body)
		goto error
	}

	collector.mustEmitMetrics(ch, &validatorResponse)
	return

error:
	ch <- prometheus.NewInvalidMetric(collector.totalValidatorsDesc, err)
	ch <- prometheus.NewInvalidMetric(collector.epochStartHeight, err)
	ch <- prometheus.NewInvalidMetric(collector.validatorStake, err)
	ch <- prometheus.NewInvalidMetric(collector.validatorExpectedBlocks, err)
	ch <- prometheus.NewInvalidMetric(collector.validatorProducedBlocks, err)
	ch <- prometheus.NewInvalidMetric(collector.validatorIsSlashed, err)
}

func main() {
	collector := NewSolanaCollector(nearRPCAddr)
	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())
	panic(http.ListenAndServe(listenAddr, nil))
}
