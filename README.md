### alpha quality.

while the base test harness excercises the code. this software hasnt been battled tested to the extent we'd be happy saying its production ready.
we're working on it. help in this area is always appreciated.

### wasi networking functionality for golang.

This is a quick and simple socket implementation for golang in wasi that doesnt try to interopt with the wider ecosystem
that currently exists. the existing ecosystem requires munging the wasip1 namespace, which causes issues with incompatibilities.

wasinet provides its own namespaced functions and _leverages_ the wasip1 namespace where necessary. primarily clock and poll functionality.

### known missing functionality.

- system TLS certificates dont work.
- many socopts are untested.

these issues will be resolved if people provide written tests to the repository exposing the issues encountered or if we run into them in our own systems.

### what kinds of PRs we'll accept.

- test cases.
- implementation of missing functionality (soc options, weird non-standard _network_ behaviors)
- we'd take PRs that fix the interopt with the wider ecosystem as well, its just the authors opinion that currently its not worth the
  effort based on the internal api's they expose.

### commercial licensing available

we understand many companies avoid GPLv3 licenses, feel free to reach out to [us](mailto:engineering@egdaemon.com) for alternative licensing for your company.

### usage

```golang
go get -u github.com/egdaemon/wasinet/wasinet@latest github.com/egdaemon/wasinet/wazeronet@latest
```

```golang
package main

import (
    "github.com/egdaemon/wasinet/wasinet"
    "github.com/egdaemon/wasinet/wasinet/autohijack"
)

func main() {
	// required because of the tls certificate resolution doesnt work in wasi environments by default.
	http.DefaultTransport = wasinet.InsecureHTTP()
	http.Get("https://www.google.com")
}
```

```golang
package example

import (
	"context"

	"github.com/egdaemon/wasinet/wasinet/wnetruntime"
	"github.com/egdaemon/wasinet/wazeronet"
)

func Wazero(runtime wazero.Runtime) wazero.HostModuleBuilder {
	return wazeronet.Module(runtime, wnetruntime.Unrestricted())
}
```

### Rationale

Due to the slow nature of committee and ecosystem politics between systems its taking too much time to have an interropt solution.

The wider ecosystem in the wild is based of wasmer edge implementation which is unnecessarily complicated to implement and runtimes like wazero
don't have the api to properly intergrate networking piecemeal in the wasi_snapshot_preview1 namespace.

Golang maintainers have little to no interest in a stopgap solution in the stdlib due to overly pedantic adherence to the go compatibility promises despite precendent for experimental features in stdlib existing in the past.

This has resulted in fragmented wasi_snapshot_preview1 implementations where you can't extend the wasi_snapshot_preview1 namespace with networking infrastructure and inconsistent behavior between the implementations even in the same langauge ecosystem.

Now, do not misunderstand us, we understand everyones position which is why we wrote this library. its meant to be a stop gap that
is as transparent as possible until the ecosystem at large gets its act together. In the meantime this library is meant to be simple to inject,
and be runtime agnostic while enabling network functionality within a guest wasi binary written in golang and using this library.

I'd like to thank the authors of [wazero](https://github.com/tetratelabs/wazero), and [dispatchrun/net](https://github.com/dispatchrun/net). both wonderful pieces of software and I shameless plumbed their depths when implementing this library.
