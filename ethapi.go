package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type EthBalanceResponse struct {
	Result string
}

type EthMarketPriceResponse struct {
	Ticker struct {
		Price  string
		Base   string
		Target string
		Volume string
		Change string
	}
	Timestamp uint64
	Success   bool
	Error     string
}

func EthBalance(address string, apiKey string) float64 {
	url := strings.Join([]string{
		"https://api.etherscan.io/api?module=account&action=balance&address=",
		address,
		"&tag=latest&apikey=",
		apiKey,
	}, "")
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Unable to fetch eth balance", err)
		return 0.0
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Unable to fetch eth balance", err)
		return 0.0
	}
	r := EthBalanceResponse{}
	json.Unmarshal(b, &r)
	v, _ := strconv.ParseFloat(r.Result, 64)
	return (v / 1000000000000000000.0)
}

func EthMarketPrice() float64 {
	resp, err := http.Get("https://api.cryptonator.com/api/ticker/eth-usd")
	if err != nil {
		fmt.Println("Unable to fetch eth market price", err)
		return 0.0
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Unable to fetch eth market price", err)
		return 0.0
	}
	r := EthMarketPriceResponse{}
	json.Unmarshal(b, &r)
	if !r.Success {
		fmt.Println("Invalid api response for eth market price")
		return 0.0
	}
	v, _ := strconv.ParseFloat(r.Ticker.Price, 64)
	return v
}
