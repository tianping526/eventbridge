package informer

import (
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

func TestInformer_WatchAndHandle(_ *testing.T) {
	_ = NewInformer(log.DefaultLogger, nil, nil)
}
