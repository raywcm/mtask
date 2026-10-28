#!/usr/bin/env python3
"""Bounded concurrent-task soak test. Example: python3 scripts/stress.py --tasks 32 --seconds 5."""
import argparse
import concurrent.futures
import json
import pathlib
import subprocess
import tempfile
import time
from integration import Harness


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary',default='bin/tdock')
    parser.add_argument('--tasks',type=int,default=32)
    parser.add_argument('--seconds',type=float,default=5)
    args=parser.parse_args()
    if not 1<=args.tasks<=256 or args.seconds<=0:parser.error('tasks must be 1..256 and seconds positive')
    with tempfile.TemporaryDirectory(prefix='tdock-stress-') as root:
        h=Harness(root,pathlib.Path(args.binary).resolve())
        try:
            with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
                tasks=list(pool.map(lambda i:h.create('soak-'+str(i),'/bin/sh','-c','while :; do printf "alive\\n"; sleep 1; done'),range(args.tasks)))
                records=json.loads(h.cli('ls','--json').stdout)
                pids=','.join(str(r['runtime']['broker_pid']) for r in records)
                def rss():
                    output=subprocess.check_output(['ps','-o','rss=','-p',pids],text=True)
                    return sum(int(line) for line in output.splitlines())
                initial_rss=rss()
                deadline=time.monotonic()+args.seconds
                observations=0
                while time.monotonic()<deadline:
                    records=json.loads(h.cli('ls','--json').stdout)
                    assert len(records)==args.tasks and all(r['health']=='online' for r in records)
                    observations+=1
                    time.sleep(.25)
                final_rss=rss()
                list(pool.map(lambda task:h.cli('stop',task),tasks))
                assert all(h.show(task)['health']=='finished' for task in tasks)
            print(f'PASS {args.tasks} live tasks, {args.seconds:g}s observation, {observations} full health scans; broker RSS {initial_rss} -> {final_rss} KiB; all brokers stopped')
        finally:h.cleanup()

if __name__=='__main__':main()
