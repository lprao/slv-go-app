package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"handler/function"

	"github.com/lprao/slv-go-lib/pkg/logger"
	handler "github.com/openfaas/templates-sdk/go-http"
)

var Logger *logger.Log

func main() {

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", 8082),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	http.HandleFunc("/", sensorHandler())
	listenUntilShutdown(s)
}

func listenUntilShutdown(s *http.Server) {
	idleConnsClosed := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM)

		<-sig

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			Logger.Errorf("Error in Shutdown: %v", err)
		}

		Logger.Infof("Exiting.")

		close(idleConnsClosed)
	}()

	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			Logger.Errorf("Error ListenAndServe: %v", err)
			close(idleConnsClosed)
		}
	}()

	<-idleConnsClosed
}

func sensorHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var input []byte

		if r.Body != nil {
			defer r.Body.Close()

			bodyBytes, bodyErr := ioutil.ReadAll(r.Body)
			if bodyErr != nil {
				Logger.Errorf("Error reading body from request.")
			}

			input = bodyBytes
		}

		req := handler.Request{
			Body:        input,
			Header:      r.Header,
			Method:      r.Method,
			QueryString: r.URL.RawQuery,
		}
		req.WithContext(r.Context())

		result, resultErr := function.StoreSoilSensorValue(req)
		if resultErr != nil {
			Logger.Infof("Failed update sensor data %v", resultErr)
		}

		if result.Header != nil {
			for k, v := range result.Header {
				w.Header()[k] = v
			}
		}

		w.WriteHeader(result.StatusCode)
		w.Write(result.Body)
	}
}

func init() {
	Logger = logger.NewLogger()
}
