// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resultset_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googlecloudplatform/pi-delivery/pkg/resultset"
	"github.com/googlecloudplatform/pi-delivery/pkg/ycd"
	"github.com/stretchr/testify/assert"
)

func TestResultSet_Sort(t *testing.T) {
	t.Parallel()
	f := func(i int64) *ycd.YCDFile {
		return &ycd.YCDFile{
			Header: &ycd.Header{
				BlockID: i,
			},
		}
	}
	set := resultset.ResultSet{
		f(2), f(0), f(1),
	}
	if got := set.Len(); got != 3 {
		t.Errorf("Len(): want = 3, got = %d", got)
	}
	testCases := []struct {
		i, j int
		want bool
	}{
		{1, 0, true},
		{0, 1, false},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Less %d %d", tc.i, tc.j), func(t *testing.T) {
			if got := set.Less(tc.i, tc.j); got != tc.want {
				t.Errorf("Less(%d, %d) = want (%v), got (%v)", tc.i, tc.j, tc.want, got)
			}
		})
	}
	sort.Sort(set)
	want := resultset.ResultSet{
		f(0), f(1), f(2),
	}
	if diff := cmp.Diff(want, set); diff != "" {
		t.Errorf("Set after Sort(): (-want, +got):\n%s", diff)
	}
}

func TestResultSet_Decimal(t *testing.T) {
	t.Parallel()
	testSet := resultset.ResultSet{
		{
			Header: &ycd.Header{
				FileVersion: "1.1.0",
				Radix:       10,
				FirstDigits: "3.14159265358979323846264338327950288419716939937510",
				TotalDigits: int64(0),
				BlockSize:   int64(100),
				BlockID:     int64(0),
				Length:      198,
			},
			Name:             "Pi - Dec - Chudnovsky/Pi - Dec - Chudnovsky - 0.ycd",
			FirstDigitOffset: 201,
		},
		{
			Header: &ycd.Header{
				FileVersion: "1.1.0",
				Radix:       10,
				FirstDigits: "3.14159265358979323846264338327950288419716939937510",
				TotalDigits: int64(0),
				BlockSize:   int64(100),
				BlockID:     int64(1),
				Length:      198,
			},
			Name:             "Pi - Dec - Chudnovsky/Pi - Dec - Chudnovsky - 1.ycd",
			FirstDigitOffset: 201,
		},
		{
			Header: &ycd.Header{
				FileVersion: "1.1.0",
				Radix:       10,
				FirstDigits: "3.14159265358979323846264338327950288419716939937510",
				TotalDigits: int64(0),
				BlockSize:   int64(100),
				BlockID:     int64(2),
				Length:      198,
			},
			Name:             "Pi - Dec - Chudnovsky/Pi - Dec - Chudnovsky - 2.ycd",
			FirstDigitOffset: 201,
		},
	}

	assert.Equal(t, int64(100), testSet.BlockSize())
	assert.Equal(t, int64(300), testSet.TotalDigits())
	assert.Equal(t, int64(48), testSet.BlockByteLength())
	assert.Equal(t, int64(144), testSet.TotalByteLength())
	assert.Equal(t, 19, testSet.DigitsPerWord())
	assert.Equal(t, 10, testSet.Radix())
	assert.Equal(t, byte('3'), testSet.FirstDigit())

	testCases := []struct {
		off, expectedId, expectedOff int64
	}{
		{0, 0, 0},
		{47, 0, 47},
		{48, 1, 0},
		{143, 2, 47},
		{144, 3, 0},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("Off %d", tc.off), func(t *testing.T) {
			bId, bOff := testSet.OffsetToBlockPos(tc.off)
			assert.Equal(t, tc.expectedId, bId)
			assert.Equal(t, tc.expectedOff, bOff)
		})
	}
}

func TestResultSet_Hexadecimal(t *testing.T) {
	t.Parallel()
	testSet := resultset.ResultSet{
		{
			Header: &ycd.Header{
				FileVersion: "1.1.0",
				Radix:       16,
				FirstDigits: "3.243f6a8885a308d313198a2e03707344a4093822299f31d008",
				TotalDigits: int64(0),
				BlockSize:   int64(100),
				BlockID:     int64(0),
				Length:      198,
			},
			Name:             "Pi - Hex - Chudnovsky/Pi - Hex - Chudnovsky - 0.ycd",
			FirstDigitOffset: 201,
		},
		{
			Header: &ycd.Header{
				FileVersion: "1.1.0",
				Radix:       16,
				FirstDigits: "3.243f6a8885a308d313198a2e03707344a4093822299f31d008",
				TotalDigits: int64(0),
				BlockSize:   int64(100),
				BlockID:     int64(1),
				Length:      198,
			},
			Name:             "Pi - Hex - Chudnovsky/Pi - Hex - Chudnovsky - 1.ycd",
			FirstDigitOffset: 201,
		},
		{
			Header: &ycd.Header{
				FileVersion: "1.1.0",
				Radix:       16,
				FirstDigits: "3.243f6a8885a308d313198a2e03707344a4093822299f31d008",
				TotalDigits: int64(0),
				BlockSize:   int64(100),
				BlockID:     int64(2),
				Length:      198,
			},
			Name:             "Pi - Hex - Chudnovsky/Pi - Hex - Chudnovsky - 2.ycd",
			FirstDigitOffset: 201,
		},
	}

	assert.Equal(t, int64(100), testSet.BlockSize())
	assert.Equal(t, int64(300), testSet.TotalDigits())
	assert.Equal(t, int64(56), testSet.BlockByteLength())
	assert.Equal(t, int64(168), testSet.TotalByteLength())
	assert.Equal(t, 16, testSet.DigitsPerWord())
	assert.Equal(t, 16, testSet.Radix())
	assert.Equal(t, byte('3'), testSet.FirstDigit())

	testCases := []struct {
		off, expectedId, expectedOff int64
	}{
		{0, 0, 0},
		{55, 0, 55},
		{56, 1, 0},
		{167, 2, 55},
		{168, 3, 0},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("Off %d", tc.off), func(t *testing.T) {
			bId, bOff := testSet.OffsetToBlockPos(tc.off)
			assert.Equal(t, tc.expectedId, bId)
			assert.Equal(t, tc.expectedOff, bOff)
		})
	}
}

func TestResultSet_DecimalPartialBlock(t *testing.T) {
	t.Parallel()
	testSet := resultset.ResultSet{
		{
			Header: &ycd.Header{
				FileVersion: "1.1.0",
				Radix:       10,
				FirstDigits: "3.14159265358979323846264338327950288419716939937510",
				TotalDigits: int64(0),
				BlockSize:   int64(100),
				BlockID:     int64(0),
				Length:      198,
			},
			Name:             "Pi - Dec - Chudnovsky/Pi - Dec - Chudnovsky - 0.ycd",
			FirstDigitOffset: 201,
		},
		{
			Header: &ycd.Header{
				FileVersion: "1.1.0",
				Radix:       10,
				FirstDigits: "3.14159265358979323846264338327950288419716939937510",
				TotalDigits: int64(150),
				BlockSize:   int64(100),
				BlockID:     int64(1),
				Length:      198,
			},
			Name:             "Pi - Dec - Chudnovsky/Pi - Dec - Chudnovsky - 1.ycd",
			FirstDigitOffset: 201,
		},
	}

	assert.Equal(t, int64(100), testSet.BlockSize())
	assert.Equal(t, int64(150), testSet.TotalDigits())
	assert.Equal(t, int64(48), testSet.BlockByteLength())
	assert.Equal(t, int64(96), testSet.TotalByteLength()) // this doesn't reflect the partial block.
	assert.Equal(t, 19, testSet.DigitsPerWord())
	assert.Equal(t, 10, testSet.Radix())
	assert.Equal(t, byte('3'), testSet.FirstDigit())
}
