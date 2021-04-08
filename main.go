package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/go-co-op/gocron"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)
import bolt "go.etcd.io/bbolt"

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
		ID         int       `json:"id"`
		CreatedAt  time.Time `json:"created_at"`
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

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func hexToF(h string) float64 {
	f, _ := strconv.ParseUint(h[2:], 16, 32)
	return math.Round(float64(f)) / math.Pow(10, 6)
}

func storeStat(conn net.Conn, db *bolt.DB) {
	r := getStat(conn)
	hrHex := r.Result.Mining.Hashrate
	host := r.Result.Host.Name
	fmt.Print(host)
	fmt.Printf("\tHashrate = %.2f MH/s\n", hexToF(hrHex))
	r.Result.CreatedAt = time.Now()
	val, err := json.Marshal(r.Result)
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("stats"))
		ns, _ := b.NextSequence()
		k := int(ns)
		return b.Put(itob(k), val)
	})
	if err != nil {
		panic("Error updating db!")
	}
}

func executeStatFetchJob(conn net.Conn, db *bolt.DB, t int) {
	s := gocron.NewScheduler(time.UTC)
	s.Every(t).Second().Do(func() {
		storeStat(conn, db)
	})
	s.StartAsync()
}

func main() {
	path := "/tmp/dione-stats.db"
	db, err := bolt.Open(path, 0666, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		panic(err)
	}
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("stats"))
		if err != nil {
			panic(err)
		}
		return nil
	})

	wrkAddr := "192.168.0.103:9033"
	conn := getConn(wrkAddr)

	executeStatFetchJob(conn, db, 5)

	fmt.Println("Press Ctrl+C to quit!")
	fmt.Scanln() // remove after implementing api server
	defer conn.Close()
	// fmt.Println("Starting API server")
	// if err := http.ListenAndServe(":8088", http.HandlerFunc(todo)); err != nil {
	//   panic(err)
	// }
}
