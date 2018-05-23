package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/google/go-containerregistry/v1/remote"

	"github.com/sclevine/packs/cf/build"
	"github.com/sclevine/packs/cf/img"
	"github.com/sclevine/packs/cf/sys"
)

var (
	refName   string
	useDaemon bool
)

func init() {
	flag.BoolVar(&useDaemon, "daemon", sys.BoolEnv("PACK_USE_DAEMON"), "inspect image in docker daemon")
}

func main() {
	flag.Parse()
	refName = flag.Arg(0)
	if flag.NArg() != 1 || refName == "" {
		sys.Exit(sys.FailCode(sys.CodeInvalidArgs, "parse arguments"))
	}
	sys.Exit(inspect())
}

func inspect() error {
	if helper, err := img.SetupCredHelper(refName); err != nil {
		return sys.FailErr(err, "setup credential helper")
	} else if helper != "" {
		fmt.Printf("Using credential helper: %s\n", helper)
	}

	store, err := img.NewRegistry(refName)
	if err != nil {
		return sys.FailErr(err, "access", refName)
	}
	image, err := store.Image()
	if err != nil {
		if rErr, ok := err.(*remote.Error); ok && len(rErr.Errors) > 0 {
			switch rErr.Errors[0].Code {
			case remote.UnauthorizedErrorCode, remote.ManifestUnknownErrorCode:
				return sys.FailCode(sys.CodeNotFound, "find", refName)
			}
		}
		return sys.FailErr(err, "get", refName)
	}
	config, err := image.ConfigFile()
	if err != nil {
		return sys.FailErr(err, "get config")
	}
	out, err := encode(config.Config.Labels)
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func encode(m map[string]string) ([]byte, error) {
	out := map[string]json.RawMessage{}
	for k, v := range m {
		switch k {
		case build.BuildLabel, build.BuildpackLabel:
			out[k] = []byte(v)
		}
	}
	return json.Marshal(out)
}
