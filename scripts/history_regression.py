#!/usr/bin/env python3
"""Verify background history expiry, using only temporary state and real processes."""
import datetime
import fcntl
import json
import os
import pathlib
import sys
import tempfile
import time
from integration import Harness,until


def age(h,task):
    ended=datetime.datetime.now(datetime.timezone.utc)-datetime.timedelta(hours=2)
    def stamp(value):return value.isoformat().replace('+00:00','Z')
    with (h.state/'catalog.lock').open('r+') as lock:
        fcntl.flock(lock,fcntl.LOCK_EX)
        for filename in ('spec.json','runtime.json'):
            path=h.state/'tasks'/task/filename
            data=json.loads(path.read_text())
            if filename=='spec.json':data['created_at']=stamp(ended-datetime.timedelta(hours=1))
            else:
                data['updated_at']=stamp(ended);data['finished_at']=stamp(ended)
                data['started_at']=stamp(ended-datetime.timedelta(minutes=1))
            staged=path.with_suffix('.staged')
            staged.write_text(json.dumps(data));staged.chmod(0o600);os.replace(staged,path)


def main():
    binary=pathlib.Path(sys.argv[1] if len(sys.argv)>1 else 'bin/tdock').resolve()
    with tempfile.TemporaryDirectory(prefix='tdock-history-test-') as root:
        h=Harness(root,binary)
        try:
            settings=json.loads(h.cli('settings','--json').stdout)
            assert settings['history_retention_seconds']==3600
            live=h.create('protected-live','/bin/sh','-c','read value')
            old=h.create('expired-history','/bin/sh','-c','exit 0');h.finished(old)
            age(h,old)
            # No frontend or CLI is running during this wait. The worker must
            # remove expired metadata AND its logs without another user command.
            until(lambda:not (h.state/'tasks'/old).exists(),seconds=9)
            assert h.show(live)['health']=='online'
            h.cli('settings','set','history-retention','off')
            kept=h.create('keep-history','/bin/sh','-c','exit 0');h.finished(kept);age(h,kept)
            time.sleep(5.3)
            assert (h.state/'tasks'/kept/'output.log').exists(), 'disabled cleanup removed history'
            h.cli('settings','set','history-retention','1s')
            fresh=h.create('short-history','/bin/sh','-c','exit 0');h.finished(fresh)
            until(lambda:not (h.state/'tasks'/fresh).exists(),seconds=9)
            until(lambda:not (h.state/'tasks'/kept).exists(),seconds=9)
            assert h.show(live)['health']=='online'
            print('PASS default one-hour history expiry, background cleanup with no frontend, disabled/re-enabled policy, adjustable duration and live-task preservation',flush=True)
        finally:h.cleanup()

if __name__=='__main__':main()
