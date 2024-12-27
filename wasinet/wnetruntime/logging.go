package wnetruntime

import (
	"log/slog"
)

var dlog = slog.Default().With(slog.String("package", "wnetruntime"))
