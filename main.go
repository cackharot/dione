package main

import (
	// "bytes"
	//	"encoding/gob"
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
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

func todo(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("todo"))
}

func makeReq(conn net.Conn, payload JsonRPCRequest, res interface{}) interface{} {
	reqB, _ := json.Marshal(payload)
	reqStr := string(reqB) + "\n"
	_, err := conn.Write([]byte(reqStr))
	if err != nil {
		fmt.Println("Write to server failed:", err.Error())
		os.Exit(1)
	}
	connbuf := bufio.NewReader(conn)
	str, err := connbuf.ReadString('\n')
	if err != nil {
		fmt.Println("Unable to read from worker api", err)
		os.Exit(1)
	}

	if len(str) > 0 {
		if err := json.Unmarshal([]byte(str), &res); err != nil {
			fmt.Println("Error to unmarshal response", str, err)
			os.Exit(1)
		}
		return &res
	}
	fmt.Println("Empty response from worker api. Not good!")
	os.Exit(1)
	return nil
}

func ping(conn net.Conn) bool {
	ping := JsonRPCRequest{
		Id:      1,
		JsonRpc: "2.0",
		Method:  "miner_ping"}

	var res PingResponse
	makeReq(conn, ping, &res)
	if res.Result == "pong" {
		return true
	}
	return false
}

func getStat(conn net.Conn) MinerStatResponse {
	statReq := JsonRPCRequest{
		Id:      1,
		JsonRpc: "2.0",
		Method:  "miner_getstatdetail"}

	var stat MinerStatResponse
	makeReq(conn, statReq, &stat)
	fmt.Println("Stat = ", stat.Result.Host.Name)
	return stat
}

func getConn(wrkAddr string) net.Conn {
	tcpAddr, err := net.ResolveTCPAddr("tcp", wrkAddr)
	if err != nil {
		println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		println("Dial failed:", err.Error())
		os.Exit(1)
	}
  return conn
}

func main() {
	wrkAddr := "192.168.0.103:9033"
  conn := getConn(wrkAddr)

	ping(conn)

	getStat(conn)

	conn.Close()
	// fmt.Println("Starting API server")
	// if err := http.ListenAndServe(":8088", http.HandlerFunc(todo)); err != nil {
	//   panic(err)
	// }
}
