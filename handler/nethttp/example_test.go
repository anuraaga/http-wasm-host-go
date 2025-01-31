package wasm

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/internal/test"
)

func Example_auth() {
	ctx := context.Background()

	// Configure and compile the WebAssembly guest binary. In this case, it is
	// an auth interceptor.
	mw, err := NewMiddleware(ctx, test.AuthWasm)
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	// Create the real request handler.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"hello\": \"world\"}")) // nolint
	})

	// Wrap this with an interceptor implemented in WebAssembly.
	wrapped, err := mw.NewHandler(ctx, next)
	if err != nil {
		log.Panicln(err)
	}

	// Start the server with the wrapped handler.
	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	// Invoke some requests, only one of which should pass
	headers := []http.Header{
		{"NotAuthorization": {"1"}},
		{"Authorization": {""}},
		{"Authorization": {"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="}},
		{"Authorization": {"0"}},
	}

	for _, header := range headers {
		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		if err != nil {
			log.Panicln(err)
		}
		req.Header = header

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Panicln(err)
		}
		resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			fmt.Println("OK")
		case http.StatusUnauthorized:
			fmt.Println("Unauthorized")
		default:
			log.Panicln("unexpected status code", resp.StatusCode)
		}
	}

	// Output:
	// Unauthorized
	// Unauthorized
	// OK
	// Unauthorized
}

func Example_log() {
	ctx := context.Background()
	logger := func(_ context.Context, msg string) { fmt.Println(msg) }

	// Configure and compile the WebAssembly guest binary. In this case, it is
	// a logging interceptor.
	mw, err := NewMiddleware(ctx, test.LogWasm, httpwasm.Logger(logger))
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	// Create the real request handler.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte("{\"hello\": \"world\"}")) // nolint
	})

	// Wrap this with an interceptor implemented in WebAssembly.
	wrapped, err := mw.NewHandler(ctx, next)
	if err != nil {
		log.Panicln(err)
	}

	// Start the server with the wrapped handler.
	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	// Make a client request and print the contents to the same logger
	resp, err := http.Get(ts.URL)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	content, _ := io.ReadAll(resp.Body)
	logger(ctx, string(content))

	// Output:
	// before
	// after
	// {"hello": "world"}
}
