package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	s "strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/silenceper/pool"

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
	return errors.New("empty response from worker api, not good")
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

func getConn(wrkAddr string) (net.Conn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", wrkAddr)
	if err != nil {
		println("ResolveTCPAddr failed:", err.Error())
		return nil, err
	}
	conn, err := net.DialTimeout("tcp", tcpAddr.String(), 5*time.Second)
	if err != nil {
		println("Dial failed:", err.Error())
		return nil, err
	}
	fmt.Println("Connected to woker at ", conn.RemoteAddr())
	return conn, nil
}

func storeStat(conn net.Conn, db *bolt.DB) error {
	r, err := getStat(conn)
	if err != nil {
		return err
	}
	res := r.Result
	hrHex := r.Result.Mining.Hashrate
	hr := hexToF(hrHex)
	host := r.Result.Host.Name
	uri := r.Result.Connection.URI
	wrkName := s.Split(s.Split(uri, "@")[0], ".")[1]

	var pwr float64 = 0.0
	var devStat = make([]DeviceStat, len(res.Devices))
	for i := 0; i < len(res.Devices); i++ {
		d := res.Devices[i]
		sen := d.Hardware.Sensors
		pwr = pwr + d.Hardware.Sensors[2]
		devStat[i] = DeviceStat{
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

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("rigs"))
		k := host + "-" + wrkName
		return b.Put([]byte(k), ws)
	})
	if err != nil {
		fmt.Println("Unable to update stats in db!", err)
		return err
	}

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("stats"))
		// ns, _ := b.NextSequence()
		// k := int(ns)
		k := time.Now().Format(time.RFC3339)
		return b.Put([]byte(k), ws)
	})
	if err != nil {
		fmt.Println("Unable to update stats in db!", err)
		return err
	}

	return nil
}

func markWorkerOffline(wrkAddr string, db *bolt.DB) {
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("rigs"))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			item := WorkerStat{}
			json.Unmarshal(v, &item)
			if item.Address != wrkAddr {
				continue
			}
			item.Connected = false
			ws, _ := json.Marshal(item)
			b.Put([]byte(k), ws)
		}
		return nil
	})
}

func executeStatFetchJob(wrkAddrs []string, db *bolt.DB, t int) {
	s := gocron.NewScheduler(time.UTC)
	for _, wrkAddr := range wrkAddrs {
		wrkAddr := wrkAddr
		pl := createPool(wrkAddr)
		s.Every(t).Second().Do(func() {
			v, err := pl.Get()
			if err != nil {
				fmt.Println("Unable to connect to "+wrkAddr, err)
				pl.Release()
				markWorkerOffline(wrkAddr, db)
				return
			}
			conn := v.(net.Conn)
			err1 := storeStat(conn, db)
			pl.Put(v)
			if err1 != nil {
				fmt.Println("Unable to store stats", err1)
				pl.Release()
				return
			}
		})
	}
	s.Every(1).Hour().Do(func() {
		db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("eth_balance"))
			val := EthMarketPrice()
			if val > 0.0 {
				b.Put([]byte("market_price"), []byte(strconv.FormatFloat(val, 'f', 10, 64)))
			}
			return nil
		})
	})
	ethAddress := getEnv("DIONE_ETH_ADDRESS", "")
	apiKey := getEnv("DIONE_ETH_API_KEY", "")
	if ethAddress != "" && apiKey != "" {
		s.Every(1).Day().Do(func() {
			db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("eth_balance"))
				val := EthBalance(ethAddress, apiKey)
				b.Put([]byte("balance"), []byte(strconv.FormatFloat(val, 'f', 10, 64)))
				return nil
			})
		})
	} else {
		fmt.Println("Not fetching ETH balance as no address & api key been provided")
		fmt.Println("Use DIONE_ETH_ADDRESS & DIONE_ETH_API_KEY env var to use this feature")
	}
	s.StartAsync()
}

func setupDb() *bolt.DB {
	path := getEnv("DIONE_DB_PATH", "/tmp/dione-stats.db")
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
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("eth_balance"))
		if err != nil {
			panic(err)
		}
		return nil
	})
	return db
}

func createPool(addr string) pool.Pool {
	factory := func() (interface{}, error) {
		return getConn(addr)
	}
	close := func(v interface{}) error { return v.(net.Conn).Close() }
	poolConfig := &pool.Config{
		InitialCap:  2,
		MaxIdle:     4,
		MaxCap:      5,
		Factory:     factory,
		Close:       close,
		IdleTimeout: 15 * time.Second,
	}
	p, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		fmt.Println("err=", err)
		os.Exit(1)
	}
	fmt.Println("Tcp conn pool created: ", p.Len())
	return p
}

func main() {
	db := setupDb()

	wrkAddrs := s.Split(getEnv("DIONE_WORKER_ADDRESS", "192.168.0.125:9033"), ",")
	executeStatFetchJob(wrkAddrs, db, 5)

	fmt.Println("Starting API server on localhost:8088")
	fmt.Println("Press Ctrl+C to quit!")
	state := &AppState{db}
	runApi(state)
}
