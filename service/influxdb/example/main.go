/*

create new client

// NOTE: this assumes you've setup a user and have setup shell env variables,
// namely INFLUX_USER/INFLUX_PWD. If not just omit Username/Password below.
_, err := client.NewHTTPClient(client.HTTPConfig{
    Addr:     "http://localhost:8086",
    Username: os.Getenv("INFLUX_USER"),
    Password: os.Getenv("INFLUX_PWD"),
})
if err != nil {
    fmt.Println("Error creating InfluxDB Client: ", err.Error())
}

*/

/*

create database

// Make client
c, err := client.NewHTTPClient(client.HTTPConfig{
    Addr: "http://localhost:8086",
})
if err != nil {
    fmt.Println("Error creating InfluxDB Client: ", err.Error())
}
defer c.Close()

q := client.NewQuery("CREATE DATABASE telegraf", "", "")
if response, err := c.Query(q); err == nil && response.Error() == nil {
    fmt.Println(response.Results)
}

*/

/*

query


// Make client
c, err := client.NewHTTPClient(client.HTTPConfig{
    Addr: "http://localhost:8086",
})
if err != nil {
    fmt.Println("Error creating InfluxDB Client: ", err.Error())
}
defer c.Close()

q := client.NewQuery("SELECT count(value) FROM shapes", "square_holes", "ns")
if response, err := c.Query(q); err == nil && response.Error() == nil {
    fmt.Println(response.Results)
}


*/

/*

write with udp client

// Make client
config := client.UDPConfig{Addr: "localhost:8089"}
c, err := client.NewUDPClient(config)
if err != nil {
    fmt.Println("Error: ", err.Error())
}
defer c.Close()

// Create a new point batch
bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
    Precision: "s",
})

// Create a point and add to batch
tags := map[string]string{"cpu": "cpu-total"}
fields := map[string]interface{}{
    "idle":   10.1,
    "system": 53.3,
    "user":   46.6,
}
pt, err := client.NewPoint("cpu_usage", tags, fields, time.Now())
if err != nil {
    fmt.Println("Error: ", err.Error())
}
bp.AddPoint(pt)

// Write the batch
c.Write(bp)


*/

/*

write with http client
// Make client
c, err := client.NewHTTPClient(client.HTTPConfig{
    Addr: "http://localhost:8086",
})
if err != nil {
    fmt.Println("Error creating InfluxDB Client: ", err.Error())
}
defer c.Close()

// Create a new point batch
bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
    Database:  "BumbleBeeTuna",
    Precision: "s",
})

// Create a point and add to batch
tags := map[string]string{"cpu": "cpu-total"}
fields := map[string]interface{}{
    "idle":   10.1,
    "system": 53.3,
    "user":   46.6,
}
pt, err := client.NewPoint("cpu_usage", tags, fields, time.Now())
if err != nil {
    fmt.Println("Error: ", err.Error())
}
bp.AddPoint(pt)

// Write the batch
c.Write(bp)

*/

/*

write 1000 points

sampleSize := 1000

// Make client
c, err := client.NewHTTPClient(client.HTTPConfig{
    Addr: "http://localhost:8086",
})
if err != nil {
    fmt.Println("Error creating InfluxDB Client: ", err.Error())
}
defer c.Close()

rand.Seed(42)

bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
    Database:  "systemstats",
    Precision: "us",
})

for i := 0; i < sampleSize; i++ {
    regions := []string{"us-west1", "us-west2", "us-west3", "us-east1"}
    tags := map[string]string{
        "cpu":    "cpu-total",
        "host":   fmt.Sprintf("host%d", rand.Intn(1000)),
        "region": regions[rand.Intn(len(regions))],
    }

    idle := rand.Float64() * 100.0
    fields := map[string]interface{}{
        "idle": idle,
        "busy": 100.0 - idle,
    }

    pt, err := client.NewPoint(
        "cpu_usage",
        tags,
        fields,
        time.Now(),
    )
    if err != nil {
        println("Error:", err.Error())
        continue
    }
    bp.AddPoint(pt)
}

err = c.Write(bp)
if err != nil {
    fmt.Println("Error: ", err.Error())
}

*/

package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/influxdata/influxdb/client"
)

/*
type Config struct {
	URL        url.URL
	UnixSocket string
	Username   string
	Password   string
	UserAgent  string
	Timeout    time.Duration
	Precision  string
	UnsafeSsl  bool
}
*/

type (
	Tags   map[string]string
	Fields map[string]interface{}
)

var (
	contries = map[int]map[string]map[string]int{
		0: map[string]map[string]int{"China": map[string]int{"Beijing": 21516000, "Shanghai": 24256800, "Tianjin": 15200000}},
		1: map[string]map[string]int{"Usa": map[string]int{"New York": 8250567, "California": 3849368, "Chicago": 2873326}},
		2: map[string]map[string]int{"Japan": map[string]int{"Tokyo": 8637098, "Kanagawa": 3697894, "Osaka": 2668586}},
		3: map[string]map[string]int{"EU": map[string]int{"London": 8673713, "Berlin": 3652956, "Madrid": 3141991}},
	}
)

func newClient(host, uname, pwd string, port int) *client.Client {
	var (
		u   *url.URL
		err error
	)
	if host != "" {
		u, err = url.Parse(fmt.Sprintf("http://%s:%d", host, port))
	} else {
		u, err = url.Parse(fmt.Sprintf("http://%s:%d", "localhost", 8086))
	}

	if err != nil {
		fmt.Println(err)
		return nil
	}

	cli, err := client.NewClient(client.Config{
		URL:      *u,
		Username: uname,
		Password: pwd,
	})
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return cli
}

func main() {

	cli := newClient("", "", "", 8086)
	if cli == nil {
		return
	}

	respTime, ver, err := cli.Ping()
	if !checkError(err) {
		fmt.Printf("ping: %fs  server version: %s\n", respTime.Seconds(), ver)
	}

	// resp, err := cli.Query(client.Query{
	// 	Command: "show databases ",
	// })
	// if !checkError(err) && resp != nil {
	// 	printResp(resp)
	// }
	// fmt.Println()
	// resp, err = cli.Query(client.Query{
	// 	Command: "drop database testdb ",
	// })
	// if !checkError(err) && resp != nil {
	// 	printResp(resp)
	// }
	// fmt.Println()

	// resp, err = cli.Query(client.Query{
	// 	Command: "create database testdb",
	// })
	// if !checkError(err) && resp != nil {
	// 	printResp(resp)
	// }
	// fmt.Println()

	// 当 db参数为空时： {"error":"database is required"}
	// 当 db 参数不存在是：  {"error":"database not found: \"testdb\""}
	for {
		write(cli)
		time.Sleep(time.Second * 10)
	}
}

// 写入数据到 influxdb
func write(cli *client.Client) {
	var (
		sampleSize = 1000
		pts        = make([]client.Point, sampleSize)
		err        error
		contryID   int
		cityID     int
		contryName string
		cityName   string
		population int
		inc        int
		resp       *client.Response
	)

	rand.Seed(13131313131)
	for i := 0; i < sampleSize; i++ {

		if (rand.Int() % 16) >= 8 {
			inc = 1
		} else {
			inc = -1
		}

		contryID = rand.Int() % 4
		cityID = rand.Int() % 3
		a := contries[contryID]
		index := 0
		for ka, va := range a {
			contryName = ka
			for kb, vb := range va {
				if index == cityID {
					cityName = kb
					population = vb
					break
				}
				index++
			}
		}

		// 取出 数据库中最近的数据 根据这个数据进行
		// select * from pop where contry = 'China' and city = 'Beijing'  order by time desc limit 1
		resp, err = cli.Query(client.Query{
			Command:  strings.Join([]string{" select * from pop where contry = '", contryName, "' and city = '", cityName, "'  order by time desc limit 1"}, ""),
			Database: "testdb",
		})

		if err != nil {
			checkError(err)
			continue
		}

		p, ok := resp.Results[0].Series[0].Values[0][0].(int)

		if ok {
			fmt.Println("not a integer")
			population = p
		}

		pts[i] = client.Point{
			Measurement: "pop",
			Tags: map[string]string{
				"contry": contryName,
				"city":   cityName,
			},
			Fields: map[string]interface{}{
				"population": population * (100 + inc*(rand.Int()%50)),
			},
			Time:      time.Now(),
			Precision: "ns",
		}
		// fmt.Println(time.Now())
	}

	bps := client.BatchPoints{
		Points:   pts,
		Database: "testdb",
	}
	_, err = cli.Write(bps)
	if err != nil {
		log.Fatal(err)
	}
}

func query(cli *client.Client) {

}

func buildProtoLine(measurement string, tags map[string]string, fields map[string]interface{}, timestamp int64) string {
	var t, f string
	for k, v := range tags {
		t = strings.Join([]string{t, ",", k, "=", v}, "")
	}
	for k, v := range fields {
		f = fmt.Sprintf("%s,%s=%v", f, k, v)
	}
	return fmt.Sprintf("%s, %s %s %d", measurement, t[1:], f[1:], timestamp)
}

func printResp(resp *client.Response) {

	if resp.Err != nil {
		fmt.Printf("[error] response. %v", resp.Err)
		return
	}

	for _, a := range resp.Results {
		for _, b := range a.Series {
			fmt.Println("Name")
			fmt.Println(b.Name)
			fmt.Println("Tags")
			fmt.Println(b.Tags)
			fmt.Println("Columns")
			fmt.Println(b.Columns)
			fmt.Println("Values")
			fmt.Println(b.Values)
		}
	}
}

func checkError(e error) bool {
	if e != nil {
		fmt.Printf("[error] %s\n", e.Error())
		return true
	}
	return false
}

func checkFatal(e error) {
	if e != nil {
		fmt.Printf("[error] %s\n", e.Error())
		os.Exit(1)
	}
}
