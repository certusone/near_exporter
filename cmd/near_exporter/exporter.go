package main

import (
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
			"near_exporter_active_validators",
			"Total number of active validators",
			nil, nil),
		epochStartHeight: prometheus.NewDesc(
			"near_exporter_epoch_start_height",
			"Current epoch's start height",
			nil, nil),
		validatorStake: prometheus.NewDesc(
			"near_exporter_validator_stake",
			"Validator's stake",
			[]string{"account_id"}, nil),
		validatorExpectedBlocks: prometheus.NewDesc(
			"near_exporter_validator_expected_blocks",
			"Validators's expected blocks",
			[]string{"account_id"}, nil),
		validatorProducedBlocks: prometheus.NewDesc(
			"near_exporter_validator_produced_blocks",
			"Validator's actual produced blocks",
			[]string{"account_id"}, nil),
		validatorIsSlashed: prometheus.NewDesc(
			"near_exporter_validator_is_slashed",
			"Whether the validator is slashed",
			[]string{"account_id"}, nil),
	}
}

func (c *nearExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalValidatorsDesc
}

func (c *nearExporter) mustEmitMetrics(ch chan<- prometheus.Metric, response *ValidatorsResponse) {
	ch <- prometheus.MustNewConstMetric(c.totalValidatorsDesc, prometheus.GaugeValue,
		float64(len(response.Result.CurrentValidators)))
	ch <- prometheus.MustNewConstMetric(c.epochStartHeight, prometheus.GaugeValue,
		float64(response.Result.EpochStartHeight))

	for _, validator := range response.Result.CurrentValidators {
		stake, err := strconv.ParseFloat(validator.Stake, 64)
		if err != nil {
			ch <- prometheus.NewInvalidMetric(c.validatorStake, fmt.Errorf("invalid stake: %s", validator.Stake))
		} else {
			ch <- prometheus.MustNewConstMetric(c.validatorStake, prometheus.GaugeValue,
				stake, validator.AccountID)
		}

		ch <- prometheus.MustNewConstMetric(c.validatorExpectedBlocks, prometheus.GaugeValue,
			float64(validator.NumExpectedBlocks), validator.AccountID)
		ch <- prometheus.MustNewConstMetric(c.validatorProducedBlocks, prometheus.GaugeValue,
			float64(validator.NumProducedBlocks), validator.AccountID)

		if validator.IsSlashed {
			ch <- prometheus.MustNewConstMetric(c.validatorIsSlashed, prometheus.GaugeValue, 1, validator.AccountID)
		} else {
			ch <- prometheus.MustNewConstMetric(c.validatorIsSlashed, prometheus.GaugeValue, 0, validator.AccountID)
		}
	}
}

func (c *nearExporter) Collect(ch chan<- prometheus.Metric) {
	err := c.collect(ch)

	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.totalValidatorsDesc, err)
		ch <- prometheus.NewInvalidMetric(c.epochStartHeight, err)
		ch <- prometheus.NewInvalidMetric(c.validatorStake, err)
		ch <- prometheus.NewInvalidMetric(c.validatorExpectedBlocks, err)
		ch <- prometheus.NewInvalidMetric(c.validatorProducedBlocks, err)
		ch <- prometheus.NewInvalidMetric(c.validatorIsSlashed, err)
	}
}

func (c *nearExporter) collect(ch chan<- prometheus.Metric) error {
	// Work around https://github.com/near/nearcore/issues/3614 by only requesting
	// validator data when the node is up-to-date.
	if syncing, err := c.checkSyncStatus(); err != nil {
		return fmt.Errorf("checkSyncStatus: %w", err)
	} else if syncing {
		return errors.New("cannot export validator metrics while the node is syncing")
	}

	validators, err := c.getValidatorInfo()
	if err != nil {
		return fmt.Errorf("getValidatorInfo: %w", err)
	}

	c.mustEmitMetrics(ch, validators)
	return nil
}

func main() {
	collector := NewSolanaCollector(nearRPCAddr)
	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())
	log.Print("RPC address ", nearRPCAddr)
	log.Print("Listening on ", listenAddr)
	panic(http.ListenAndServe(listenAddr, nil))
}
