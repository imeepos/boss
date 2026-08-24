package app

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

func etlScanInterval() time.Duration {
	if raw := os.Getenv("ETL_AUTODISPATCH_INTERVAL_SECONDS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 5 * time.Minute
}

func startETLOverdueLoop(a *Application) func() {
	if a == nil || a.ETLScanner == nil {
		return func() {}
	}
	stop := make(chan struct{})
	var once sync.Once
	ticker := time.NewTicker(etlScanInterval())
	go func() {
		for {
			select {
			case <-ticker.C:
				if _, err := a.ETLScanner.Scan(context.Background()); err != nil {
					log.Printf("etl overdue dispatch: %v", err)
				}
			case <-stop:
				return
			}
		}
	}()
	return func() {
		once.Do(func() { ticker.Stop(); close(stop) })
	}
}
