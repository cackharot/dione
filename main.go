package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	s "strings"
	"time"

	"github.com/go-co-op/gocron"

	bolt "go.etcd.io/bbolt"
)

func makeReq(conn net.Conn, payload JsonRPCRequest, res interface{}) error {
	reqB, _ := json.Marshal(payload)
	reqStr := string(reqB) + "\n"
	_, err := conn.Write([]byte(reqStr))
	if err != nil {
		fmt.Println("Write to server failed:", err.Error())
		return err
	}
	connbuf := bufio.NewReader(conn)
	str, err := connbuf.ReadString('\n')
	if err != nil {
		fmt.Println("Unable to read from worker api", err)
		return err
	}

	if len(str) > 0 {
		if err := json.Unmarshal([]byte(str), &res); err != nil {
			fmt.Println("Error to unmarshal response", str, err)
			return err
		}
		return nil
	}
	fmt.Println("Empty response from worker api. Not good!")
	return errors.New("Empty response from worker api. Not good!")
}

func ping(conn net.Conn) bool {
	ping := JsonRPCRequest{
		Id:      1,
		JsonRpc: "2.0",
		Method:  "miner_ping"}

	var res PingResponse
	if err := makeReq(conn, ping, &res); err != nil {
		return false
	}
	return res.Result == "pong"
}

func getStat(conn net.Conn) (*MinerStatResponse, error) {
	statReq := JsonRPCRequest{
		Id:      1,
		JsonRpc: "2.0",
		Method:  "miner_getstatdetail"}

	var stat MinerStatResponse
	if err := makeReq(conn, statReq, &stat); err != nil {
		return nil, err
	}
	return &stat, nil
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
	fmt.Println("Connected to woker at ", conn.RemoteAddr())
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
	r, err := getStat(conn)
	if err != nil {
		return
	}
	res := r.Result
	hrHex := r.Result.Mining.Hashrate
	hr := hexToF(hrHex)
	host := r.Result.Host.Name
	uri := r.Result.Connection.URI
	wrkName := s.Split(s.Split(uri, "@")[0], ".")[1]
	fmt.Print(host + "\t" + wrkName)
	fmt.Printf("\tHashrate = %.2f MH/s\n", hr)
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
	var pwr float64 = 0.0
	var devStat = make([]DeviceStat, len(res.Devices))
	for i := 0; i < len(res.Devices); i++ {
		d := res.Devices[i]
		sen := d.Hardware.Sensors
		pwr = pwr + d.Hardware.Sensors[2]
		devStat[0] = DeviceStat{
			Id:          d.Index,
			Device_type: d.Hardware.Type,
			Mode:        d.Mode,
			Name:        d.Hardware.Name,
			Hashrate:    hexToF(d.Mining.Hashrate),
			Paused:      d.Mining.Paused,
			Shares:      d.Mining.Shares,
			Temperature: sen[0],
			Fan:         sen[1],
			Power:       sen[2],
		}
	}
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("rigs"))
		k := host + "-" + wrkName
		ws, _ := json.Marshal(WorkerStat{
			Name:       wrkName,
			Hostname:   host,
			Address:    conn.RemoteAddr().String(),
			Connected:  res.Connection.Connected,
			URI:        uri,
			Runtime:    float64(res.Host.Runtime),
			Hashrate:   hr,
			Difficulty: res.Mining.Difficulty,
			Shares:     res.Mining.Shares,
			Devices:    devStat,
			Power:      pwr,
		})
		return b.Put([]byte(k), ws)
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
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("rigs"))
		if err != nil {
			panic(err)
		}
		return nil
	})

	wrkAddr := "192.168.0.103:9033"
	conn := getConn(wrkAddr)

	executeStatFetchJob(conn, db, 5)

	fmt.Println("Starting API server on localhost:8088")
	fmt.Println("Press Ctrl+C to quit!")
	state := &AppState{db}
	runApi(state)
	defer conn.Close()
}
