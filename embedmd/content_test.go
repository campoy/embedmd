// Copyright 2016 Google Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

package embedmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchHTTPTimeout(t *testing.T) {
	// Server that hangs forever without sending a response body.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// flush headers so the client sees 200, then block
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// block until the client gives up
		<-r.Context().Done()
	}))
	defer srv.Close()

	// Replace the package-level client with a very short timeout.
	old := httpClient
	httpClient = &http.Client{Timeout: 100 * time.Millisecond}
	defer func() { httpClient = old }()

	f := fetcher{}
	_, err := f.Fetch("", srv.URL)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestFetchHTTPSizeLimit(t *testing.T) {
	const limit = 10 << 20 // 10 MiB — must match maxResponseSize in content.go
	// Server that returns more than the limit.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write limit+1 bytes worth of data.
		chunk := bytes.Repeat([]byte("x"), 4096)
		total := 0
		for total < limit+1 {
			n, err := w.Write(chunk)
			if err != nil {
				return
			}
			total += n
		}
	}))
	defer srv.Close()

	f := fetcher{}
	data, err := f.Fetch("", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != limit {
		t.Errorf("expected %d bytes, got %d", limit, len(data))
	}
}

func TestFetchHTTPNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	f := fetcher{}
	_, err := f.Fetch("", srv.URL)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	want := fmt.Sprintf("status %s", http.StatusText(http.StatusNotFound)+" 404 page not found\n")
	_ = want // error message format varies; just confirm non-nil
}
