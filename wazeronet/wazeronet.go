package wazeronet

import (
	"github.com/egdaemon/wasinet/wasisocket"
	"github.com/tetratelabs/wazero"
)

func Wazero(host wazero.HostModuleBuilder) wazero.HostModuleBuilder {
	return host.NewFunctionBuilder().
		WithFunc(wasisocket.Open).Export("socket_open")
}
