# Build and package

Requires Go 1.26+ and make. Runtime needs only the binary.

```sh
make build
./scripts/install.sh --prefix "$HOME/.local" ./bin/mtask
# System installation:
sudo ./scripts/install.sh --prefix /usr/local ./bin/mtask
```

The installer uses atomic replacement and leaves existing brokers running.
It does not change shell configuration.

## Packages

Use GoReleaser v2.19.0 and nFPM v2.48.0 from a Git checkout with a commit:

```sh
goreleaser check
goreleaser release --snapshot --clean --skip=publish
python3 scripts/check_packages.py dist
```

`--clean` removes the old `dist` directory. Builds disable CGO and target
Linux/macOS × amd64/arm64. Output includes four archives, Linux Deb/RPM packages,
SHA-256 checksums, documentation and licenses. Public releases remain disabled.

For an existing Linux binary:

```sh
MTASK_VERSION=0.4.4-dev MTASK_ARCH=amd64 MTASK_BINARY=./bin/mtask-linux-amd64 \
  nfpm package --config packaging/nfpm.yaml --packager deb --target dist/
```

Packages install `/usr/bin/mtask` and `/usr/share/doc/mtask`. Upgrade/removal does
not stop tasks or install a global service.

## Local APT index

```sh
python3 scripts/apt_repo.py dist dist/apt
```

This creates Packages/Release files for amd64/arm64. A trusted local test source
can use `deb [trusted=yes] file:/absolute/path/to/dist/apt ./`. A public repository
needs hosting, signing keys and InRelease/Release.gpg. None is published yet.

See [testing](stability.md) and [compatibility](compatibility.md) for verification
commands and platform coverage.
