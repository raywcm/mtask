#!/usr/bin/env python3
"""Check local archives, Deb/RPM metadata/payloads and SHA-256 sums without installing."""
import gzip
import hashlib
import io
import pathlib
import struct
import sys
import tarfile
from apt_repo import control_fields

NOTICE_COUNT = sum(p.is_file() for p in (pathlib.Path(__file__).resolve().parent.parent/'licenses').rglob('*'))
assert NOTICE_COUNT > 0


def check_elf(binary,arch):
    assert binary[:4]==b'\x7fELF'
    assert struct.unpack('<H',binary[18:20])[0]==(183 if arch=='arm64' else 62)


def deb(package):
    arch='arm64' if 'arm64' in package.name else 'amd64'
    control=control_fields(package)
    assert 'Package: tdock' in control and 'Architecture: '+arch in control
    data=package.read_bytes();offset=8;found=False
    while offset+60<=len(data):
        header=data[offset:offset+60];name=header[:16].decode().strip().rstrip('/')
        size=int(header[48:58]);offset+=60
        member=data[offset:offset+size];offset+=size+(size%2)
        if name.startswith('data.tar'):
            with tarfile.open(fileobj=io.BytesIO(member)) as archive:
                members={m.name.lstrip('./'):m for m in archive.getmembers()}
                assert members['usr/bin/tdock'].mode&0o777==0o755
                assert 'usr/share/doc/tdock/docs/usage.md' in members
                assert 'usr/share/doc/tdock/docs/development-log.md' not in members
                assert 'usr/share/doc/tdock/README.zh-CN.md' not in members
                assert 'usr/share/doc/tdock/LICENSE' in members
                assert 'usr/share/doc/tdock/THIRD_PARTY_NOTICES.md' in members
                assert len([n for n in members if n.startswith('usr/share/doc/tdock/licenses/') and members[n].isfile()]) == NOTICE_COUNT
                assert 'usr/share/doc/tdock/docs/archive/architecture-a0.md' in members
                check_elf(archive.extractfile(members['usr/bin/tdock']).read(),arch)
                found=True
    assert found


def rpm(package):
    arch='arm64' if 'arm64' in package.name else 'amd64'
    data=package.read_bytes()
    assert data[:4]==b'\xed\xab\xee\xdb'
    def header(offset):
        assert data[offset:offset+3]==b'\x8e\xad\xe8'
        count,length=struct.unpack('>II',data[offset+8:offset+16]);base=offset+16+count*16
        tags={}
        for i in range(count):
            tag,kind,start,_=struct.unpack('>IIII',data[offset+16+i*16:offset+32+i*16])
            if kind==6:tags[tag]=data[base+start:].split(b'\0',1)[0].decode()
        return tags,base+length
    _,end=header(96)
    tags,end=header((end+7)//8*8)
    assert tags[1000]=='tdock' and tags[1022]==('aarch64' if arch=='arm64' else 'x86_64')
    assert tags[1125]=='gzip'
    payload=gzip.decompress(data[end:]);offset=0;found=False;names=set()
    while offset+110<=len(payload):
        assert payload[offset:offset+6] in (b'070701',b'070702')
        fields=[int(payload[offset+6+8*i:offset+14+8*i],16) for i in range(13)]
        size,namesize=fields[6],fields[11]
        name=payload[offset+110:offset+110+namesize-1].decode()
        if fields[1] & 0o170000 == 0o100000:names.add(name.lstrip('./'))
        start=(offset+110+namesize+3)//4*4
        if name.lstrip('./')=='usr/bin/tdock':
            assert fields[1]&0o777==0o755
            check_elf(payload[start:start+size],arch);found=True
        offset=(start+size+3)//4*4
        if name=='TRAILER!!!':break
    assert found
    assert {'usr/share/doc/tdock/LICENSE','usr/share/doc/tdock/THIRD_PARTY_NOTICES.md'} <= names
    assert 'usr/share/doc/tdock/docs/development-log.md' not in names
    assert 'usr/share/doc/tdock/README.zh-CN.md' not in names
    assert len([n for n in names if n.startswith('usr/share/doc/tdock/licenses/')]) == NOTICE_COUNT


def main():
    root=pathlib.Path(sys.argv[1] if len(sys.argv)>1 else 'dist')
    checked=0
    for line in (root/'checksums.txt').read_text().splitlines():
        digest,name=line.split(None,1)
        assert hashlib.sha256((root/name.strip()).read_bytes()).hexdigest()==digest
    for path in sorted(root.glob('*.deb')):deb(path);checked+=1
    for path in sorted(root.glob('*.rpm')):rpm(path);checked+=1
    for path in sorted(root.glob('*.tar.gz')):
        with tarfile.open(path) as archive:
            names={m.name for m in archive.getmembers()}
            assert {'tdock','install.sh','README.md','LICENSE','THIRD_PARTY_NOTICES.md'} <= names
            assert len([m for m in archive.getmembers() if m.name.startswith('licenses/') and m.isfile()]) == NOTICE_COUNT
            assert 'docs/archive/architecture-a0.md' in names
            assert 'docs/development-log.md' not in names
            assert 'README.zh-CN.md' not in names
        checked+=1
    assert checked==8,checked
    print('PASS 8 artifacts: archives, Deb/RPM names, modes, architectures, ELF payloads, MIT/original dependency notices and checksums')

if __name__=='__main__':main()
