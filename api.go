package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"

	bolt "go.etcd.io/bbolt"
)

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
	r.SetFuncMap(fm)
	r.Static("/assets", "./assets")
	r.StaticFile("/favicon.ico", "./assets/img/favicon.ico")
	r.LoadHTMLGlob("templates/*")
	r.GET("/", func(c *gin.Context) {
		lst := make([]WorkerStat, 0)
		state.db.View(func(tx *bolt.Tx) error {
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
			fmt.Println(v.Shares)
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
	})
	r.Run(":8080")
}
