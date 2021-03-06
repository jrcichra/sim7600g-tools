package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joncrlsn/dque"
	"github.com/stratoberry/go-gpsd"
)

const url = "https://ingest.jrcichra.dev"
const database = "public"
const table = "gps"
const TPVInterval = 10

//gps record with hostname metadata
type dbRecord struct {
	*gpsd.TPVReport
	Hostname string `json:"hostname"`
}

//gpsRecord - a basic GPS datapoint
type gpsRecord struct {
	Value gpsd.TPVReport
}

// dbRecordBuilder - abstracts out a dbRecord for dque to work
func dbRecordBuilder() interface{} {
	return &dbRecord{}
}

func makeQueue() *dque.DQue {
	qName := "gps_queue"
	qDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	log.Println("qDir=", qDir)
	segmentSize := 50
	if err != nil {
		panic(err)
	}
	q, err := dque.NewOrOpen(qName, qDir, segmentSize, dbRecordBuilder)
	if err != nil {
		panic(err)
	}
	return q
}

func makeGPS(hostname string, port int) *gpsd.Session {
	gps, err := gpsd.Dial(hostname + ":" + strconv.Itoa(port))
	if err != nil {
		panic(err)
	}
	return gps
}

func dbrToBytes(dbr *dbRecord) []byte {
	b, err := json.Marshal(dbr)
	if err != nil {
		panic(err)
	}
	return b
}

func dbrToMap(dbr *dbRecord) map[string]interface{} {
	m := make(map[string]interface{})
	b, err := json.Marshal(dbr)

	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &m)
	if err != nil {
		panic(err)
	}
	return m
}

func queueToPost(q *dque.DQue, h *http.Client) {
	for {
		// Only dequeue if we could successfully POST
		var t interface{}
		var err error

		if t, err = q.PeekBlock(); err != nil {
			if err != dque.ErrEmpty {
				log.Fatal("Error dequeuing item ", err)
			}
		}

		// rename columns
		record := t.(*dbRecord)
		m := dbrToMap(record)
		tme := m["time"]
		delete(m, "time")
		m["gps_timestamp"] = tme
		b, err := json.Marshal(m)
		if err != nil {
			log.Println(err)
			continue
		}

		// parse time to Golang time for checks

		ttime, err := time.Parse(time.RFC3339, tme.(string))
		if err != nil {
			log.Println(err)
			continue
		}

		//Make sure lat/lon isn't zero & the time is at least reasonable. If it is bad data, skip it
		// log.Println("spew of m:")
		// spew.Dump(m)
		if m["lat"] == 0.0 || m["lon"] == 0.0 || ttime.Year() < time.Now().Year()-1 {
			// Dequeue this variable and skip
			log.Println("Found a bad lat+lon, skipping entry and dequeuing it")
			_, err = q.Dequeue()
			if err != nil {
				panic(err)
			}
			continue
		}

		log.Println("POSTING:", string(b))
		req, err := http.NewRequest("POST", url+"/"+database+"/"+table, bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			log.Println(err)
			continue
		}

		resp, err := h.Do(req)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Println("response Status:", resp.Status)
		// body, _ := ioutil.ReadAll(resp.Body)
		// log.Println("response Body:", string(body))
		if resp.StatusCode == 200 {
			// Dequeue this variable now
			_, err = q.Dequeue()
			if err != nil {
				panic(err)
			}
		} else {
			log.Println("Not dequeuing because I didn't get a 200 OK")
		}
		resp.Body.Close()
	}
}

func main() {
	//connect to gps
	gps := makeGPS("localhost", 2947)
	q := makeQueue()
	h := &http.Client{}
	hostname, err := os.Hostname()
	log.Println("hostname=", hostname)
	if err != nil {
		panic(err)
	}
	// Handle sending off HTTP posts
	go queueToPost(q, h)
	// Ticker for only one TPV per interval
	ticker := time.NewTicker(TPVInterval * time.Second)
	// GPS loop
	gps.AddFilter("TPV", func(r interface{}) {
		// This anon function is called every time a new TPV value comes in, scoped this way so we can use q easily
		select {
		case <-ticker.C:
			// Only enqueue if the ticker went off
			tpv := r.(*gpsd.TPVReport)
			dbr := &dbRecord{tpv, hostname}
			err := q.Enqueue(dbr)
			if err != nil {
				panic(err)
			}
		default:
			// Do nothing with the data
		}
		// log.Println("Location inserted into file queue:", string(dbrToBytes(dbr)))
	})
	done := gps.Watch()
	<-done
	log.Println("The program is ending because gps.Watch() came back. Hopefully this shouldn't happen")
}
