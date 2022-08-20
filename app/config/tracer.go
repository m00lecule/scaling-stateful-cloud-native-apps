package config

import (
	"go.opentelemetry.io/otel"
)

var Tracer = otel.Tracer("gin-server")
