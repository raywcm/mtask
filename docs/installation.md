# Installation and releases

## One-line install

```sh
curl -fsSL https://github.com/raywcm/mtask/releases/latest/download/install.sh | sh
```

The installer detects Linux/macOS and amd64/arm64, verifies SHA-256, and installs
to `~/.local/bin`. Add that directory to PATH. Run it again to upgrade; existing
tasks stay running. It requires curl, tar, and sha256sum or shasum.

To inspect the script or choose a version/directory:

```sh
curl -fsSLo install.sh https://github.com/raywcm/mtask/releases/latest/download/install.sh
less install.sh
sh install.sh --version v0.4.4 --prefix "$HOME/.local"
# System installation:
sudo sh install.sh --prefix /usr/local
```

## Packages

Download the matching archive, `.deb` or `.rpm` and `checksums.txt` from
[GitHub Releases](https://github.com/raywcm/mtask/releases). Verify the checksum
before installing. Debian/Ubuntu users can install a downloaded package with
`sudo apt install ./mtask_0.4.4_linux_amd64.deb` (use arm64 on ARM hosts).
There is no public APT repository yet.

## Build from source

Requires Go 1.26+ and make. Runtime needs only the binary.

```sh
make build
./scripts/install.sh --prefix "$HOME/.local" ./bin/mtask
# System installation:
sudo ./scripts/install.sh --prefix /usr/local ./bin/mtask
```

The installer uses atomic replacement and leaves existing brokers running.
It does not change shell configuration.

## Build release packages

Use GoReleaser v2.19.0 and nFPM v2.48.0 from a Git checkout with a commit:

```sh
goreleaser check
goreleaser release --snapshot --clean --skip=publish
python3 scripts/check_packages.py dist
```

`--clean` removes the old `dist` directory. Builds disable CGO and target
Linux/macOS × amd64/arm64. Output includes four archives, Linux Deb/RPM packages,
SHA-256 checksums, documentation and licenses.

Pushing a `vX.Y.Z` tag runs the Linux/macOS tests and package-install checks, then
builds a draft release. Its packages are checked before publication. Add release
notes at `docs/releases/vX.Y.Z.md` before tagging. The workflow uses GitHub's
temporary token; no personal token is needed.

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
