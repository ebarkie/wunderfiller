// Copyright 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package wxcalc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDewPoint(t *testing.T) {
	assert.Equal(t, 72.75063875457386, DewPoint(88.44, 60), "Dew point")
}
