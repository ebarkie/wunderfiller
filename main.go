// Copyright 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Wunderground filler.
package main

//go:generate ./version.sh

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ebarkie/weatherlink/calc"
	"github.com/ebarkie/weatherlink/data"
	"github.com/ebarkie/wunderground"
)

const (
	dateLayout = "2006-01-02"
	timeLayout = "2006-01-02 15:04 MST"
)

var (
	errNoPasswd = errors.New("password is needed to upload")
)

func archiveInterval(archive []data.Archive) time.Duration {
	if len(archive) > 1 {
		return archive[0].Timestamp.Sub(archive[1].Timestamp)
	}

	return 5 * time.Minute
}

func getArchive(serverAddress string, begin, end time.Time) (archive []data.Archive, err error) {
	// Build HTTP request.
	req, _ := http.NewRequest("GET", "http://"+serverAddress+":8080/archive", nil)

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

func fill(begin, end time.Time, addr, id, password string, test bool) error {
	fmt.Printf("Fill range is %s to %s.\n", begin.Format(timeLayout), end.Format(timeLayout))

	// Get records from archive.
	archive, err := getArchive(addr, begin, end)
	if err != nil {
		return err
	}
	interval := archiveInterval(archive)
	fmt.Printf("Found %d archive records.\n", len(archive))

	// Get timestamps from Wunderground.
	wuTimes, err := getWuTimes(begin, end, id)
	if err != nil {
		return err
	}
	fmt.Printf("Found %d Wunderground observations.\n", len(wuTimes))

	// Loop through archive records and perform uploads for anything
	// that's missing.
	//
	// Each archive record contains the rain accomulation for its period so
	// by replaying them in time order (the reverse of what we get) we can
	// keep a daily accumulator.
	var daily struct {
		rainAccum float64
		day       int
	}
	for i := len(archive) - 1; i >= 0; i-- {
		a := archive[i]

		if a.Timestamp.Day() != daily.day {
			daily.day = a.Timestamp.Day()
			daily.rainAccum = 0
		}
		daily.rainAccum += a.RainAccum

		if fuzzyTimeMatch(a.Timestamp, wuTimes) {
			continue
		}

		fmt.Printf("\tMissing %s: ", a.Timestamp.Format(timeLayout))
		if test {
			fmt.Println("not uploaded.")
		} else {
			err := upload(id, password, interval, daily.rainAccum, a)
			if err != nil {
				fmt.Printf("%s.\n", err.Error())
			} else {
				fmt.Println("successfully uploaded.")
			}
		}

	}

	fmt.Println("Done!")
	return nil
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

func getWuTimes(begin, end time.Time, id string) ([]time.Time, error) {
	var ts []time.Time

	w := wunderground.Pws{ID: id}
	for date := begin; date.Before(end.Add(1 * time.Second)); date = date.AddDate(0, 0, 1) {
		obs, err := w.DownloadDaily(date)
		if err != nil {
			return ts, err
		}

		for _, o := range obs {
			ts = append(ts, o.Time.Time)
		}
	}

	return ts, nil
}

func upload(id, password string, interval time.Duration, dailyRain float64, a data.Archive) error {
	if password == "" {
		return errNoPasswd
	}

	w := wunderground.Pws{
		ID:           id,
		Password:     password,
		SoftwareType: "GoWunder wf." + version,
		Interval:     interval,
		Time:         a.Timestamp,
	}

	wx := &wunderground.Wx{}
	wx.Barometer(a.Bar)
	wx.DailyRain(dailyRain)
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

	return w.Upload(wx)
}

func main() {
	defBegin := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
	defBeginStr := defBegin.Format(dateLayout)
	defEndStr := defBegin.AddDate(0, 0, 1).Format(dateLayout)

	beginStr := flag.String("begin", defBeginStr, "fill begin date YYYY-MM-DD")
	endStr := flag.String("end", defEndStr, "fill begin date YYYY-MM-DD")
	addr := flag.String("server", "", "weather server address (REQUIRED)")
	id := flag.String("id", "", "personal weather station id (REQUIRED)")
	password := flag.String("pass", "", "personal weather station password")
	test := flag.Bool("test", false, "test only/do not upload")

	flag.Parse()

	var begin, end time.Time
	var err error
	for _, d := range []struct {
		t *time.Time
		s string
	}{
		{&begin, *beginStr},
		{&end, *endStr},
	} {
		*d.t, err = time.ParseInLocation(dateLayout, d.s, time.Local)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if *addr == "" || *id == "" {
		flag.Usage()
		return
	}

	err = fill(begin, end, *addr, *id, *password, *test)
	if err != nil {
		fmt.Println(err)
	}
}
