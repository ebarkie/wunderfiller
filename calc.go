// Copyright 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"math"
)

func cToF(c float64) float64 {
	return c*1.8 + 32.0
}

func fToC(f float64) float64 {
	return (f - 32.0) * 5.0 / 9.0
}

func dewPoint(tf float64, h int) float64 {
	// Magnus-Tetens formula.

	const a = 17.27
	const b = 237.7

	tc := fToC(tf)
	x := (a*tc)/(b+tc) + math.Log(float64(h)/100.0)
	dpc := (b * x) / (a - x)

	return cToF(dpc)
}
