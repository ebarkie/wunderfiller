![Push](https://github.com/ebarkie/wunderfiller/workflows/Push/badge.svg)

# Wunderground Filler

Uses the Davis station archive data to fill in any data gaps in Weather
Underground station data.

## Installation

```
$ go generate
$ go build
```

## Usage

```
Usage of ./wunderfiller:
  -begin string
        fill begin date YYYY-MM-DD (default "2018-08-26")
  -end string
        fill end date YYYY-MM-DD (default "2018-08-26")
  -id string
        personal weather station id (REQUIRED)
  -pass string
        personal weather station password
  -station string
        weather station address (REQUIRED)
  -test
        test only/do not upload

$ ./wunderfiller -station wx:8080 -id Kxxyyyynn -pass deadbeef -test
```

## License

Copyright (c) 2016-2019 Eric Barkie. All rights reserved.  
Use of this source code is governed by the MIT license
that can be found in the [LICENSE](LICENSE) file.
