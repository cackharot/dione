package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	bolt "go.etcd.io/bbolt"
)

//go:embed assets/css/* assets/img/* assets/fonts/* assets/js/* templates/*
var f embed.FS

func runApi(state *AppState) {
	r := gin.Default()
	fm := template.FuncMap{
		"divide": func(a, b float64) float64 {
			return a / b
		},
		"inr": func(amt float64) float64 {
			return amt * 72.0
		},
		"multiply": func(a, b float64) float64 {
			return a * b
		},
	}
	templ := template.Must(template.New("").Funcs(fm).ParseFS(f, "templates/*.tmpl"))
	r.SetHTMLTemplate(templ)
	r.SetFuncMap(fm)

	r.StaticFS("/public", http.FS(f))
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.FileFromFS("assets/img/favicon.ico", http.FS(f))
	})

	r.GET("/", func(c *gin.Context) {
		dashboard(c, state.db)
	})

	r.GET("/stats", func(c *gin.Context) {
		calc_stats(c, state.db)
	})

	r.Run(":8080")
}

type ChartData struct {
	Hashrates      map[int64]float64
	Power          map[int64]float64
	Workers        map[int64]int
	ValidShares    map[int64]int
	InvalidShares  map[int64]int
	RejectedShares map[int64]int
}

func updateData(k []byte, v []byte, wrks []string, data *ChartData) {
	item := WorkerStat{}
	json.Unmarshal(v, &item)
	key, err := time.Parse(time.RFC3339, string(k))
	if err != nil {
		println("error in parsing time"+string(k), err)
	} else {
		t := key.Unix() * 1000
		if item.Connected {
			data.Hashrates[t] = item.Hashrate
			data.Workers[t] = len(wrks)
			data.Power[t] = item.Power
			data.ValidShares[t] = item.Shares[0]
			data.InvalidShares[t] = item.Shares[1]
			data.RejectedShares[t] = item.Shares[2]
		}
	}
}

func calc_stats(c *gin.Context, db *bolt.DB) {
	data := &ChartData{
		Hashrates:      make(map[int64]float64),
		Power:          make(map[int64]float64),
		Workers:        make(map[int64]int),
		ValidShares:    make(map[int64]int),
		InvalidShares:  make(map[int64]int),
		RejectedShares: make(map[int64]int),
	}
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("stats"))
		rigsBk := tx.Bucket([]byte("rigs"))
		wrks := make([]string, 0)
		rigsBk.ForEach(func(k, v []byte) error {
			item := WorkerStat{}
			json.Unmarshal(v, &item)
			wrks = append(wrks, item.Name)
			return nil
		})

		c := b.Cursor()
		tn := time.Now()
		min := []byte(tn.AddDate(0, 0, -1).Format(time.RFC3339))
		max := []byte(tn.Add(time.Duration(10) * time.Minute).Format(time.RFC3339))
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			updateData(k, v, wrks, data)
		}

		return nil
	})

	var d int64 = 1000 * 60 * 60 // 1 hr resample
	var min time.Time = time.Now().Local().AddDate(0, 0, -1)
	data.Hashrates = dfResample(data.Hashrates, min, d)
	data.Power = dfResample(data.Power, min, d)
	data.Workers = dfResampleInt(data.Workers, min, d)
	data.ValidShares = dfResampleInt(data.ValidShares, min, d)
	data.InvalidShares = dfResampleInt(data.InvalidShares, min, d)
	data.RejectedShares = dfResampleInt(data.RejectedShares, min, d)

	c.JSON(200, data)
}

func dashboard(c *gin.Context, db *bolt.DB) {
	lst := make([]WorkerStat, 0)
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("rigs"))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			item := WorkerStat{}
			json.Unmarshal(v, &item)
			if len(item.Name) <= 0 {
				continue
			}
			lst = append(lst, item)
		}
		return nil
	})
	gs := GlobalStat{
		Rigs:            len(lst),
		Workers:         len(lst),
		ActiveWorkers:   len(lst),
		InactiveWorkers: 0,
		PowerCostKwh:    0.1,
	}
	gs.Shares = make([]int, 4)
	for _, v := range lst {
		gs.Hashrate = gs.Hashrate + v.Hashrate
		gs.Power = gs.Power + v.Power
		gs.Devices = gs.Devices + len(v.Devices)
		gs.Temps = make(map[string]map[string]float64)
		for _, d := range v.Devices {
			if gs.Temps[v.Name] == nil {
				gs.Temps[v.Name] = make(map[string]float64)
			}
			gs.Temps[v.Name][d.Name] = d.Temperature
		}
		for sidx, s := range v.Shares {
			gs.Shares[sidx] = gs.Shares[sidx] + s
		}
	}
	gs.PowerCost = 24.0 / (1000 / gs.Power) * gs.PowerCostKwh

	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Miner Stats",
		"items": lst,
		"stat":  gs,
	})
}
