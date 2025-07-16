package data

import (
	"github.com/google/wire"

	"github.com/tianping526/eventbridge/app/internal/event"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewLogger,
	NewConfig,
	NewConfigBootstrap,
	NewRegistrar,
	NewMetric,
	NewTracerProvider,
	NewEntClient,
	NewRules,
	NewBusReflector,
	NewBuses,
	NewReceiver,
	NewSender,
	NewEventRepo,
)

func NewReceiver(b Bus) event.Receiver {
	return b
}

func NewSender(b Bus) Sender {
	return b
}
