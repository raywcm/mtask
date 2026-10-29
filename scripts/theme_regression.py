#!/usr/bin/env python3
"""Real terminal background replies: automatic palettes and monitor input isolation."""
import os,pathlib,sys,tempfile
from integration import ENV,Harness,until,wait_output
from screen_regression import Terminal
from ui_regression import wait_view
from monitor_control_regression import FIXTURE,received


def main():
 binary=pathlib.Path(sys.argv[1] if len(sys.argv)>1 else 'bin/tdock').resolve()
 saved=dict(ENV)
 ENV.pop('NO_COLOR',None);ENV.pop('COLORFGBG',None);ENV['COLORTERM']='truecolor'
 with tempfile.TemporaryDirectory(prefix='tdock-theme-') as root:
  h=Harness(root,binary)
  try:
   h.cli('settings','set','language','en');h.cli('settings','set','history-retention','off')
   ref=h.create('theme-work',sys.executable,'-c',FIXTURE,'theme')
   for preference in ['auto','light']:
    h.cli('settings','set','theme',preference)
    pid,fd=h.spawn_tty();t=Terminal()
    t.feed(wait_output(fd,b'\x1b]11;?'))
    os.write(fd,b'\x1b]11;rgb:ffff/ffff/ffff\a')
    wait_view(fd,t,lambda s:'theme-work' in s and t.current.buffer[0][2].fg=='006c70')
    assert t.current.buffer[0][2].bg=='f6f8fc','automatic and explicit Light differ'
    os.write(fd,b'v');wait_view(fd,t,lambda s:'READY-theme' in s and 'Control' in s)
    # Delayed/split terminal replies belong to the manager, never to the task.
    reply=b'\x1b]11;rgb:ffff/ffff/ffff\a'
    os.write(fd,reply[:7]);__import__('time').sleep(.08);os.write(fd,reply[7:])
    os.write(fd,b'safe-input');until(lambda:received(h,ref).endswith(b'safe-input'))
    assert b'\x1b]11;' not in received(h,ref)
    os.write(fd,b'\x1b[24~');wait_view(fd,t,lambda s:'Memory' in s and 'Control' not in s)
    os.write(fd,b'q');assert h.join(pid)==0
   print('PASS automatic Light matches explicit Light, real OSC background replies, delayed/split replies isolated from monitor input')
  finally:h.cleanup();ENV.clear();ENV.update(saved)
if __name__=='__main__':main()
