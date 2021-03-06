// Copyright 2020 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sqlx

import (
	"bytes"
	"database/sql/driver"
	"sync"
	"time"

	"github.com/xgfone/cast"
)

// BufferDefaultCap is the default capacity to be allocated for buffer from pool.
var BufferDefaultCap = 64

var bufpool = sync.Pool{New: func() interface{} {
	b := new(bytes.Buffer)
	b.Grow(BufferDefaultCap)
	return b
}}

func getBuffer() *bytes.Buffer    { return bufpool.Get().(*bytes.Buffer) }
func putBuffer(buf *bytes.Buffer) { buf.Reset(); bufpool.Put(buf) }

var slicepool = sync.Pool{New: func() interface{} {
	return make([]interface{}, 0, ArgsDefaultCap)
}}

func getSlice() []interface{}   { return slicepool.Get().([]interface{}) }
func putSlice(ss []interface{}) { ss = ss[:0]; slicepool.Put(ss) }

// Time is used to read/write the time.Time from/to DB.
type Time struct {
	time.Time
}

// Now returns the current Time.
func Now() Time { return Time{Time: time.Now().In(Location)} }

// Value implements the interface driver.Valuer.
func (t Time) Value() (driver.Value, error) { return t.Time, nil }

// Scan implements the interface sql.Scanner.
func (t *Time) Scan(src interface{}) (err error) {
	_t, err := cast.ToTimeInLocation(Location, src, DatetimeLayout)
	if err == nil {
		t.Time = _t
	}
	return
}

// Bool is used to read/write the BOOLEAN from/to DB.
type Bool bool

// Value implements the interface driver.Valuer.
func (b Bool) Value() (driver.Value, error) { return bool(b), nil }

// Scan implements the interface sql.Scanner.
func (b *Bool) Scan(src interface{}) (err error) {
	_b, err := cast.ToBool(src)
	if err == nil {
		*b = Bool(_b)
	}
	return
}
