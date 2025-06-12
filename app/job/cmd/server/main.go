package main

import (
	"flag"
	_ "net/http/pprof"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/http"
	_ "go.uber.org/automaxprocs"

	"github.com/tianping526/eventbridge/app/internal/event"
	"github.com/tianping526/eventbridge/app/job/internal/conf"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name = "eventbridge.job"
	// Version is the version of the compiled software.
	Version string
	// flagConf is the config flag.
	flagConf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagConf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, rr registry.Registrar, hs *http.Server, es *event.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			hs,
			es,
		),
		kratos.Registrar(rr),
	)
}

func main() {
	flag.Parse()

	var appInfo conf.AppInfo
	appInfo.Id = id
	appInfo.Name = Name
	appInfo.Version = Version
	appInfo.FlagConf = flagConf
	app, cleanup, err := wireApp(&appInfo)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err = app.Run(); err != nil {
		panic(err)
	}
}
