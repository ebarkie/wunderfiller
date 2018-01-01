# Wunderground Filler

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](http://choosealicense.com/licenses/mit/)
[![Build Status](https://travis-ci.org/ebarkie/wunderfiller.svg?branch=master)](https://travis-ci.org/ebarkie/wunderfiller)

Uses the Davis station archive data to fill in any data gaps in Weather
Underground station data.

## Installation

```
$ go get
$ go generate
$ go build
```

## Usage

```
Usage of ./wunderfiller:
  -begin string
    	fill begin date YYYY-MM-DD (default "2018-01-01")
  -end string
    	fill begin date YYYY-MM-DD (default "2018-01-02")
  -id string
    	personal weather station id (REQUIRED)
  -pass string
    	personal weather station password
  -server string
    	weather server address (REQUIRED)
  -test
    	test only/do not upload

$ ./wunderfiller -server wx:8080 -id Kxxyyyynn -pass deadbeef -test
```

## License

Copyright (c) 2016-2018 Eric Barkie. All rights reserved.  
Use of this source code is governed by the MIT license
that can be found in the [LICENSE](LICENSE) file.
