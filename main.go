// Copyright 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Wunderground filler.
package main

//go:generate ./version.sh

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ebarkie/weatherlink/calc"
	"github.com/ebarkie/weatherlink/data"
	"github.com/ebarkie/wunderground"
)

type filler struct {
	dailyRain float64

	id       string
	interval time.Duration
	password string
}

func archiveInterval(archive []data.Archive) (interval time.Duration) {
	if len(archive) > 1 {
		interval = archive[0].Timestamp.Sub(archive[1].Timestamp)
	} else {
		interval = 5 * time.Minute
	}

	return
}

func archiveRecords(serverAddress string, begin time.Time, end time.Time) (archive []data.Archive, err error) {
	// Build HTTP request.
	req, _ := http.NewRequest("GET", "http://"+serverAddress+"/archive", nil)

	// Create GET query parameters.
	q := req.URL.Query()
	q.Add("begin", begin.Format(time.RFC3339))
	q.Add("end", end.Format(time.RFC3339))
	req.URL.RawQuery = q.Encode()

	// Initiate HTTP request.
	client := &http.Client{}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Check HTTP return status code.
	if resp.StatusCode != 200 {
		err = fmt.Errorf("HTTP request returned non-200 status code %d", resp.StatusCode)
		return
	}

	// Parse response.
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &archive)

	return
}

func fill(begin time.Time, serverAddress string, id string, password string, test bool) {
	end := begin.Add(86399 * time.Second)
	fmt.Printf("Checking range %s to %s.\n", begin, end)

	// Get records from archive.
	archive, err := archiveRecords(serverAddress, begin, end)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("\tFound %d archive records.\n", len(archive))

	// Get timestamps from Wunderground.
	wuTimes, err := wuDailyTimes(begin, id)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("\tFound %d Wunderground records.\n", len(wuTimes))

	// Loop through archive records and perform uploads for anything
	// that's missing.
	//
	// Each record only includes the rain accumulation for the archive
	// period but if replay them in time order (the reverse of what we
	// get) then we can calculate the daily rain.
	f := filler{
		id:       id,
		password: password,
		interval: archiveInterval(archive),
	}
	for i := len(archive) - 1; i >= 0; i-- {
		a := archive[i]
		f.dailyRain += a.RainAccum

		if fuzzyTimeMatch(a.Timestamp, wuTimes) {
			continue
		}

		fmt.Printf("\tMissing %s: ", a.Timestamp)
		if test {
			fmt.Println("not uploaded.")
		} else {
			err := f.wuUpload(a)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("successfully uploaded.")
			}
		}

	}
}

func fuzzyTimeMatch(a time.Time, times []time.Time) bool {
	const splay = 150 * time.Second // 2.5m

	for _, t := range times {
		min := a.Add(-splay)
		max := a.Add(splay)
		if t.After(min) && t.Before(max) {
			return true
		}
	}

	return false
}

func wuDailyTimes(day time.Time, id string) (times []time.Time, err error) {
	w := wunderground.Pws{ID: id}
	obs, err := w.DownloadDaily(day)
	if err == nil {
		for _, o := range obs {
			times = append(times, o.Time.Time)
		}
	}

	return
}

func (f *filler) wuUpload(a data.Archive) (err error) {
	w := wunderground.Pws{ID: f.id, Password: f.password}
	w.SoftwareType = "GoWunder wf." + version
	w.Interval = f.interval

	w.Time = a.Timestamp
	wx := &wunderground.Wx{}
	wx.Barometer(a.Bar)
	wx.DailyRain(f.dailyRain)
	wx.DewPoint(calc.DewPoint(a.OutTemp, a.OutHumidity))
	wx.OutdoorHumidity(a.OutHumidity)
	wx.OutdoorTemperature(a.OutTemp)
	wx.RainRate(a.RainRateHi)
	for _, v := range a.SoilMoist {
		if v != nil {
			wx.SoilMoisture(*v)
		}
	}
	for _, v := range a.SoilTemp {
		if v != nil {
			wx.SoilTemperature(float64(*v))
		}
	}
	wx.SolarRadiation(a.SolarRad)
	wx.UVIndex(a.UVIndexAvg)
	wx.WindDirection(a.WindDirPrevail)
	wx.WindSpeed(float64(a.WindSpeedAvg))
	if a.WindSpeedHi > a.WindSpeedAvg {
		wx.WindGustDirection(a.WindDirHi)
		wx.WindGustSpeed(float64(a.WindSpeedHi))
	}

	err = w.Upload(wx)

	return
}

func main() {
	date := flag.String("date", "", "date to fill YYYY-MM-DD")
	id := flag.String("id", "", "personal weather station id (REQUIRED)")
	password := flag.String("pass", "", "personal weather station password (REQUIRED)")
	serverAddress := flag.String("server", "", "weather server address (REQUIRED)")
	test := flag.Bool("test", false, "test only/do not upload")

	flag.Parse()

	begin := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
	if *date != "" {
		begin, _ = time.ParseInLocation("2006-01-02", *date, time.Local)
	}

	if begin.IsZero() ||
		(len(*id) == 0) ||
		(len(*password) == 0) ||
		(len(*serverAddress) == 0) {
		flag.Usage()
	} else {
		fill(begin, *serverAddress, *id, *password, *test)
		fmt.Println("Done!")
	}
}
