#!/usr/bin/env python3
"""Create a local, flat APT repository; never upload or publish it.

Usage: python3 scripts/apt_repo.py DEB_DIRECTORY OUTPUT_DIRECTORY
For a future public repository, sign Release with a separately managed GPG key.
"""
import datetime
import gzip
import hashlib
import io
import pathlib
import shutil
import sys
import tarfile


def control_fields(package):
    data=package.read_bytes()
    if data[:8]!=b'!<arch>\n':raise ValueError('not a Debian ar archive')
    offset=8
    while offset+60<=len(data):
        header=data[offset:offset+60]
        name=header[:16].decode().strip().rstrip('/')
        size=int(header[48:58]);offset+=60
        member=data[offset:offset+size];offset+=size+(size%2)
        if name.startswith('control.tar'):
            with tarfile.open(fileobj=io.BytesIO(member)) as archive:
                for info in archive.getmembers():
                    if info.name.lstrip('./')=='control':
                        return archive.extractfile(info).read().decode().strip()
    raise ValueError('missing control member')


def build(source,destination):
    destination.mkdir(parents=True,exist_ok=True)
    packages=[]
    for package in sorted(source.glob('*.deb')):
        target=destination/package.name
        if package.resolve()!=target.resolve():shutil.copy2(package,target)
        data=target.read_bytes()
        packages.append(control_fields(target)+'\nFilename: ./'+target.name+'\nSize: '+str(len(data))+'\nSHA256: '+hashlib.sha256(data).hexdigest()+'\n')
    if not packages:raise ValueError('no Debian packages found')
    index=('\n'.join(packages)+'\n').encode()
    (destination/'Packages').write_bytes(index)
    (destination/'Packages.gz').write_bytes(gzip.compress(index,mtime=0))
    release=['Origin: mtask local build','Label: mtask','Suite: local','Codename: local',
             'Date: '+datetime.datetime.now(datetime.timezone.utc).strftime('%a, %d %b %Y %H:%M:%S +0000'),
             'Architectures: amd64 arm64','Description: Local mtask package repository','SHA256:']
    for name in ('Packages','Packages.gz'):
        data=(destination/name).read_bytes()
        release.append(' '+hashlib.sha256(data).hexdigest()+' '+str(len(data))+' '+name)
    (destination/'Release').write_text('\n'.join(release)+'\n')
    print(f'Local APT repository: {destination} ({len(packages)} packages; unsigned; not published)')

if __name__=='__main__':
    if len(sys.argv)!=3:raise SystemExit(__doc__)
    build(pathlib.Path(sys.argv[1]).resolve(),pathlib.Path(sys.argv[2]).resolve())
