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
	"database/sql"

	"github.com/xgfone/cast"
)

// Datetime is the time layout format of SQL DATETIME
const Datetime = "2006-01-02 15:04:05"

// Scanner is a interface to scan and return the value.
type Scanner interface {
	sql.Scanner
	Value() interface{}
}

// NewScanner returns a new Scanner.
func NewScanner(scan func(src interface{}) (dst interface{}, err error)) Scanner {
	return &scanner{scan: scan}
}

type scanner struct {
	value interface{}
	scan  func(src interface{}) (dst interface{}, err error)
}

func (s *scanner) Value() interface{} { return s.value }
func (s *scanner) Scan(src interface{}) error {
	dst, err := s.scan(src)
	if err == nil {
		s.value = dst
	}
	return err
}

/// --------------------------------------------------------------------------

// IntScanner returns a scanner to scan the source to int.
func IntScanner() Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		return cast.ToInt(src)
	})
}

// Int32Scanner returns a scanner to scan the source to int32.
func Int32Scanner() Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		return cast.ToInt32(src)
	})
}

// Int64Scanner returns a scanner to scan the source to int64.
func Int64Scanner() Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		return cast.ToInt64(src)
	})
}

// UintScanner returns a scanner to scan the source to uint.
func UintScanner() Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		return cast.ToUint(src)
	})
}

// Uint32Scanner returns a scanner to scan the source to int32.
func Uint32Scanner() Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		return cast.ToUint32(src)
	})
}

// Uint64Scanner returns a scanner to scan the source to int64.
func Uint64Scanner() Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		return cast.ToUint64(src)
	})
}

// Float64Scanner returns a scanner to scan the source to float64.
func Float64Scanner() Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		return cast.ToFloat64(src)
	})
}

// BoolScanner returns a scanner to scan the source to bool.
func BoolScanner() Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		if bs, ok := src.([]byte); ok {
			switch len(bs) {
			case 0:
				return false, nil
			case 1:
				return bs[0] != 0, nil
			}
		}
		return cast.ToBool(src)
	})
}

// StringScanner returns a scanner to scan the source to string.
func StringScanner() Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		return cast.ToString(src)
	})
}

// DurationScanner returns a scanner to scan the source to time.Duration.
func DurationScanner() Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		return cast.ToDuration(src)
	})
}

// TimeScanner returns a scanner to scan the source to time.Time.
func TimeScanner(layout ...string) Scanner {
	return NewScanner(func(src interface{}) (dst interface{}, err error) {
		return cast.ToTime(src, layout...)
	})
}