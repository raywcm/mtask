#!/usr/bin/env python3
"""Offline release-installer checks; all downloads and prefixes are temporary."""
import hashlib
import io
import os
import pathlib
import shutil
import subprocess
import sys
import tarfile
import tempfile

ROOT = pathlib.Path(__file__).resolve().parent.parent
INSTALLER = ROOT / 'install.sh'
VERSION = '0.4.4'
BINARY = b'#!/bin/sh\nprintf "MTasK v0.4.4\\n"\n'


def fixture(root, system, machine, hash_command='sha256sum', failure=''):
    tools = root / 'tools'
    tools.mkdir()
    for name in ['tar', 'gzip', 'awk', 'mktemp', 'mkdir', 'rm', 'cp', 'chmod', 'mv', hash_command]:
        source = shutil.which(name)
        if not source:
            raise RuntimeError('missing test tool: ' + name)
        (tools / name).symlink_to(source)
    (tools / 'uname').write_text('#!/bin/sh\ncase "$1" in -s) echo ' + system + ';; -m) echo ' + machine + ';; esac\n')
    (tools / 'uname').chmod(0o755)
    platform = {'Linux': 'linux', 'Darwin': 'darwin'}.get(system, 'other')
    arch = {'x86_64': 'amd64', 'aarch64': 'arm64', 'arm64': 'arm64'}.get(machine, 'other')
    archive = root / f'mtask_{VERSION}_{platform}_{arch}.tar.gz'
    with tarfile.open(archive, 'w:gz') as bundle:
        member = tarfile.TarInfo('mtask')
        member.mode = 0o755
        member.size = len(BINARY)
        bundle.addfile(member, io.BytesIO(BINARY))
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    if failure == 'checksum':
        digest = '0' * 64
    checksum = root / 'checksums.txt'
    checksum.write_text(digest + '  ' + archive.name + '\n')
    mock = tools / 'curl'
    mock.write_text('#!' + sys.executable + '\n' + '''
import os,pathlib,shutil,sys
args=sys.argv[1:];url=args[-1]
assert args[args.index('--proto')+1]=='=https'
assert args[args.index('--proto-redir')+1]=='=https'
root=pathlib.Path(os.environ['FIXTURE'])
with (root/'requests').open('a') as log:log.write(url+'\\n')
if url.endswith('/releases/latest'):
    print('https://github.com/raywcm/mtask/releases/tag/v0.4.4',end='')
else:
    source=root/url.rsplit('/',1)[1]
    if os.environ['FAILURE']=='download':sys.exit(22)
    shutil.copyfile(source,args[args.index('--output')+1])
''')
    mock.chmod(0o755)
    prefix = root / 'prefix with spaces'
    (prefix / 'bin').mkdir(parents=True)
    installed = prefix / 'bin/mtask'
    installed.write_bytes(b'previous-installation')
    temporary = root / 'tmp'
    temporary.mkdir()
    env = dict(os.environ, PATH=str(tools), FIXTURE=str(root), FAILURE=failure, TMPDIR=str(temporary))
    return prefix, installed, temporary, env


def main():
    for system, machine in [('Linux', 'x86_64'), ('Linux', 'aarch64'), ('Darwin', 'x86_64'), ('Darwin', 'arm64')]:
        for hash_command in ['sha256sum', 'shasum']:
            if not shutil.which(hash_command):
                continue
            with tempfile.TemporaryDirectory(prefix='mtask-installer-') as directory:
                root = pathlib.Path(directory)
                prefix, installed, temporary, env = fixture(root, system, machine, hash_command)
                result = subprocess.run(['/bin/sh', str(INSTALLER), '--prefix', str(prefix)], env=env, capture_output=True)
                assert result.returncode == 0, result.stderr
                assert installed.read_bytes() == BINARY and installed.stat().st_mode & 0o777 == 0o755
                assert not list(temporary.iterdir()) and not list((prefix/'bin').glob('.mtask.*'))
                requests = (root/'requests').read_text()
                assert '/releases/latest' in requests and '/releases/download/v0.4.4/' in requests
                # A pinned version skips latest-release discovery and can upgrade in place.
                (root/'requests').unlink()
                result = subprocess.run(['/bin/sh', str(INSTALLER), '--version', VERSION, '--prefix', str(prefix)], env=env, capture_output=True)
                assert result.returncode == 0, result.stderr
                assert '/releases/latest' not in (root/'requests').read_text()

    for failure in ['checksum', 'download']:
        with tempfile.TemporaryDirectory(prefix='mtask-installer-fail-') as directory:
            root = pathlib.Path(directory)
            hash_command = 'sha256sum' if shutil.which('sha256sum') else 'shasum'
            prefix, installed, temporary, env = fixture(root, 'Linux', 'x86_64', hash_command, failure)
            result = subprocess.run(['/bin/sh', str(INSTALLER), '--version', VERSION, '--prefix', str(prefix)], env=env, capture_output=True)
            assert result.returncode != 0
            assert installed.read_bytes() == b'previous-installation'
            assert not list(temporary.iterdir()) and not list((prefix/'bin').glob('.mtask.*'))

    with tempfile.TemporaryDirectory(prefix='mtask-installer-platform-') as directory:
        root = pathlib.Path(directory)
        hash_command = 'sha256sum' if shutil.which('sha256sum') else 'shasum'
        prefix, installed, _, env = fixture(root, 'FreeBSD', 'x86_64', hash_command)
        result = subprocess.run(['/bin/sh', str(INSTALLER), '--prefix', str(prefix)], env=env, capture_output=True)
        assert result.returncode != 0 and not (root/'requests').exists()
        assert installed.read_bytes() == b'previous-installation'
    print('PASS installer: four targets, latest/pinned versions, both SHA-256 tools, atomic upgrade, failed-download/checksum preservation and unsupported OS')


if __name__ == '__main__':
    main()
