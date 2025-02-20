package informer

import (
	"errors"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-kratos/kratos/v2/log"
)

func TestInformer_WatchAndHandle(_ *testing.T) {
	_ = NewInformer(log.DefaultLogger, nil, nil)
}

func TestRetry(t *testing.T) {
	cnt := 0
	err := backoff.Retry(func() error {
		cnt++
		if cnt < 3 {
			return errors.New("error")
		}
		return backoff.Permanent(NewReflectorClosedError())
	}, backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(24*time.Hour)))
	if err != nil {
		if !IsReflectorClosedError(err) {
			t.Errorf("expect reflector closed error, got %v", err)
		}
	}
	if cnt != 3 {
		t.Errorf("expect 3 retries, got %d", cnt)
	}
}
