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

package rest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/goccy/go-json"
	"github.com/googlecloudplatform/pi-delivery/gen/index"
	"github.com/googlecloudplatform/pi-delivery/pkg/service"
	"go.ajitem.com/zapdriver"
	"go.uber.org/zap"
)

var _serv *service.Service
var _servOnce sync.Once

var maxDigitsPerRequest = 1000
var bucketName = index.BucketName

const (
	envMaxDigitsPerRequest = "PI_MAX_DIGITS_PER_REQUEST"
	envBucketName          = "PI_BUCKET_NAME"
)

func init() {
	functions.HTTP("Get", Get)
	functions.HTTP("NotFound", NotFound)
	logger, err := zapdriver.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "zapdriver.NewProduction() failed: %v", err)
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)
	defer logger.Sync()
	log := logger.Sugar()

	// Read configurations from env.
	if s := os.Getenv(envMaxDigitsPerRequest); s != "" {
		if i, err := strconv.Atoi(s); err != nil {
			log.Error("invalid env value", "name", envMaxDigitsPerRequest, "value", s)
		} else {
			maxDigitsPerRequest = i
		}
	}
	if s := os.Getenv(envBucketName); s != "" {
		bucketName = s
	}
	log.Info("Config",
		"maxDigitsPerRequest", maxDigitsPerRequest,
		"bucketName", bucketName,
	)
}

func getService(ctx context.Context, l *zap.Logger) *service.Service {
	_servOnce.Do(func() {
		serv, err := service.NewService(ctx, bucketName)
		if err != nil {
			l.Fatal("service.NewService() failed", zap.Error(err))
		}
		_serv = serv
	})
	return _serv
}

func namedLogger(l *zap.Logger, name string, req *http.Request) *zap.Logger {
	return l.Named(name).
		With(
			zapdriver.HTTP(zapdriver.NewHTTP(req, nil)),
		)
}

func writeError(l *zap.Logger, res http.ResponseWriter, code int, s string) {
	l.Error(s, zap.Int("code", code))
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(code)
	_, err := io.WriteString(res, s)
	if err != nil {
		l.Error("WriteString failed", zap.Error(err))
	}
}

func getIntQueryParam(l *zap.Logger, q url.Values, name string, def int64) (int64, error) {
	// TODO(yuryu): Use Has() when go 1.17 is available on Functions.
	p := q.Get(name)
	if p == "" {
		return def, nil
	}
	i, err := strconv.ParseInt(p, 10, 64)
	if err != nil {
		l.Error("ParseInt failed", zap.Error(err), zap.String("param", name), zap.String("value", p))
		return 0, fmt.Errorf("invalid request: %s", name)
	}
	return i, nil
}

// GetResponse is the JSON response for Get.
type GetResponse struct {
	// Content is a string representation of Pi digits.
	// ex. "31415926535897932384626433832795028841971693993"
	Content string `json:"content"`
}

// Get is the entrypoint for the API.
// It takes three parameters in the query string:
//   - start (int64): the digit position to read from.
//   - numberOfDigits(int64): number of digits to read.
//   - radix (int): the radix of pi to read. 10 or 16. default 10.
//
// It returns a JSON response as GetResponse.
func Get(res http.ResponseWriter, req *http.Request) {
	l := namedLogger(zap.L(), "Get", req)
	defer l.Sync()

	l.Debug("Get start")
	res.Header().Set("Access-Control-Allow-Origin", "*")

	q := req.URL.Query()
	radix, err := getIntQueryParam(l, q, "radix", 10)
	if err != nil {
		writeError(l, res, http.StatusBadRequest, err.Error())
		return
	}
	if radix != 10 && radix != 16 {
		writeError(l, res, http.StatusBadRequest, "radix must be either 10 or 16")
		return
	}
	set := index.Decimal
	if radix == 16 {
		set = index.Hexadecimal
	}

	start, err := getIntQueryParam(l, q, "start", 0)
	if err != nil {
		writeError(l, res, http.StatusBadRequest, err.Error())
		return
	}
	if start < 0 {
		writeError(l, res, http.StatusBadRequest, "start is negative")
		return
	}
	if start > set.TotalDigits() {
		writeError(l, res, http.StatusBadRequest, "start out of range")
		return
	}

	numberOfDigits, err := getIntQueryParam(l, q, "numberOfDigits", 100)
	if err != nil {
		writeError(l, res, http.StatusBadRequest, err.Error())
		return
	}
	if numberOfDigits < 0 {
		writeError(l, res, http.StatusBadRequest, "numberOfDigits is negative")
		return
	}
	if numberOfDigits > int64(maxDigitsPerRequest) {
		writeError(l, res, http.StatusBadRequest, "numberOfDigits is too big")
		return
	}

	unpacked, err := getService(req.Context(), l).
		Get(req.Context(), set, start, numberOfDigits)
	if err != nil {
		l.Error("Get() failed", zap.Error(err))
		writeError(l, res, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).EncodeWithOption(
		&GetResponse{Content: string(unpacked)},
		json.DisableHTMLEscape(),
	)
	if err != nil {
		l.Error("json encode failed", zap.Error(err))
	}
}

// NotFound returns 404 for all requests.
// This is necessary because LB can't return 404 by itself.
// https://issuetracker.google.com/160192483
func NotFound(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusNotFound)
	io.WriteString(res, fmt.Sprintf("The requested url %s was not found.\n", req.URL.Path))
}
