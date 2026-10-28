#!/usr/bin/env python3
"""APT install/reinstall/remove check for a disposable Linux CI runner only."""
import json
import os
import pathlib
import subprocess
import tempfile


def main():
    if os.environ.get('CI')!='true':raise SystemExit('Run this system-package test only in disposable CI (CI=true).')
    root=pathlib.Path.cwd()
    native=root/'dist/mtask_linux_amd64_v1/mtask'
    with tempfile.TemporaryDirectory(prefix='mtask-apt-test-') as temp:
        temp=pathlib.Path(temp);os.chmod(temp,0o755)
        source=temp/'source.list';source.write_text('deb [trusted=yes] file:'+str(root/'dist/apt')+' ./\n')
        apt=['sudo','apt-get','-o','Dir::Etc::sourcelist='+str(source),'-o','Dir::Etc::sourceparts=-','-o','APT::Get::List-Cleanup=0']
        subprocess.run(apt+['update'],check=True)
        subprocess.run(apt+['install','-y','mtask'],check=True)
        state=temp/'state'
        def cli(binary,*args):
            return subprocess.check_output([str(binary),'--state-dir',str(state),*args],text=True)
        task=json.loads(cli('/usr/bin/mtask','new','-d','--json','--name','upgrade','--','/bin/sh','-c','read value'))
        taskid=task['spec']['id'];pid=task['runtime']['broker_pid']
        try:
            subprocess.run(apt+['install','--reinstall','-y','mtask'],check=True)
            assert json.loads(cli('/usr/bin/mtask','show',taskid))['runtime']['broker_pid']==pid
            subprocess.run(apt+['remove','-y','mtask'],check=True)
            after=json.loads(cli(native,'show',taskid))
            assert after['runtime']['broker_pid']==pid and after['health']=='online'
        finally:
            cli(native,'stop',taskid);cli(native,'delete',taskid)
    print('PASS local apt install/reinstall/uninstall preserves live broker identity')

if __name__=='__main__':main()
