package data

import (
	"github.com/google/wire"
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
	NewRedisCmd,
	NewSchemaLocalCache,
	NewReflector,
	NewSender,
	NewBusRepo,
	NewEventRepo,
	NewRuleRepo,
)
