package main

import (
	"context"
	"log"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/egenv"
	"github.com/egdaemon/eg/runtime/wasi/eggit"
	"github.com/egdaemon/eg/runtime/wasi/shell"
	"github.com/egdaemon/eg/runtime/x/wasi/eggolang"
)

func DNSDebug(ctx context.Context, _ eg.Op) (err error) {
	privileged := shell.Runtime().Privileged()
	return shell.Run(
		ctx,
		privileged.Newf("systemctl status systemd-resolved.service"),
		privileged.New("ss -t -l -n -p"),
	)
}

func main() {
	ctx, done := context.WithTimeout(context.Background(), egenv.TTL())
	defer done()

	err := eg.Perform(
		ctx,
		eggit.AutoClone,
		DNSDebug,
		// eg.Build(eg.DefaultModule()),
		// eg.Module(
		// 	ctx,
		// eg.DefaultModule(),
		eggolang.AutoCompile(),
		eggolang.AutoTest(),
		// ),
	)

	if err != nil {
		log.Fatalln(err)
	}
}
