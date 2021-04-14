package main

import (
	bolt "go.etcd.io/bbolt"
)

type JsonRPCRequest struct {
	Id      int    `json:"id"`
	JsonRpc string `json:"jsonrpc"`
	Method  string `json:"method"`
}

type PingResponse struct {
	Id      int    `json:"id"`
	JsonRpc string `json:"jsonrpc"`
	Result  string `json:"result"`
}

type MinerStatResponse struct {
	ID      int    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		ID         int `json:"id"`
		Connection struct {
			Connected bool   `json:"connected"`
			Switches  int    `json:"switches"`
			URI       string `json:"uri"`
		} `json:"connection"`
		Devices []struct {
			Index    int    `json:"_index"`
			Mode     string `json:"_mode"`
			Hardware struct {
				Name    string    `json:"name"`
				Pci     string    `json:"pci"`
				Sensors []float64 `json:"sensors"`
				Type    string    `json:"type"`
			} `json:"hardware"`
			Mining struct {
				Hashrate    string      `json:"hashrate"`
				PauseReason interface{} `json:"pause_reason"`
				Paused      bool        `json:"paused"`
				Segment     []string    `json:"segment"`
				Shares      []int       `json:"shares"`
			} `json:"mining"`
		} `json:"devices"`
		Host struct {
			Name    string `json:"name"`
			Runtime int    `json:"runtime"`
			Version string `json:"version"`
		} `json:"host"`
		Mining struct {
			Difficulty   float64 `json:"difficulty"`
			Epoch        int     `json:"epoch"`
			EpochChanges int     `json:"epoch_changes"`
			Hashrate     string  `json:"hashrate"`
			Shares       []int   `json:"shares"`
		} `json:"mining"`
		Monitors struct {
			Temperatures []int `json:"temperatures"`
		} `json:"monitors"`
	} `json:"result"`
}

type WorkerStat struct {
	Name       string
	Hostname   string
	Connected  bool
	Address    string
	URI        string
	Runtime    float64
	Hashrate   float64
	Difficulty float64
	Shares     []int
	Devices    []DeviceStat
	Power      float64
}

type DeviceStat struct {
	Id          int
	Device_type string
	Mode        string
	Name        string
	Hashrate    float64
	Paused      bool
	Shares      []int
	Temperature float64
	Fan         float64
	Power       float64
}

type GlobalStat struct {
	Rigs            int
	ActiveWorkers   int
	InactiveWorkers int
	Workers         int
	Devices         int
	Hashrate        float64
	Power           float64 // per 24 hr
	PowerCostKwh    float64
	PowerCost       float64
	Shares          []int
	Unpaid          Estimate
	Earnings        Earnings
}

type Earnings struct {
	Day   Estimate
	Week  Estimate
	Month Estimate
}

type Estimate struct {
	Eth float32
	Btc float32
	Usd float32
}

type AppState struct {
	db *bolt.DB
}

type DF map[int64]struct {
	len int
	sum float64
}
