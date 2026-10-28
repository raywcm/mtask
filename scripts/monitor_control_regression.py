#!/usr/bin/env python3
"""Real-PTY embedded control, opaque input, manager prefixes and task identity."""
import json,os,pathlib,re,sys,tempfile,time,shutil
from integration import Harness,until,drain,wait_output
from screen_regression import Terminal
from ui_regression import wait_view
from monitor_dialog_regression import latest_size
FIXTURE=r'''
import os,tty,signal
name=__import__('sys').argv[1]
tty.setraw(0)
def paint(*args):
 s=os.get_terminal_size(0);os.write(1,('READY-%s\r\nSIZE:%d,%d\r\n'%(name,s.lines,s.columns)).encode())
signal.signal(signal.SIGWINCH,paint);paint()
while True:
 p=os.read(0,4096)
 if not p:break
 os.write(1,b'INPUT:'+p.hex().encode()+b'\r\n')
'''
def received(h,ref):return b''.join(bytes.fromhex(p.decode()) for p in re.findall(rb'INPUT:([0-9a-f]+)',h.cli('logs',ref,'--raw').stdout))
def main():
 binary=pathlib.Path(sys.argv[1] if len(sys.argv)>1 else 'bin/mtask').resolve()
 with tempfile.TemporaryDirectory(prefix='mtask-monitor-control-') as root:
  h=Harness(root,binary)
  try:
   h.cli('settings','set','language','en');h.cli('settings','set','history-retention','off')
   a=h.create('alpha',sys.executable,'-c',FIXTURE,'alpha');b=h.create('beta',sys.executable,'-c',FIXTURE,'beta')
   until(lambda:b'READY-alpha' in h.cli('logs',a).stdout and b'READY-beta' in h.cli('logs',b).stdout)
   before={ref:h.show(ref)['runtime'] for ref in [a,b]}
   pid,fd=h.spawn_tty();t=Terminal();wait_view(fd,t,lambda s:'alpha' in s and 'Memory' in s)
   os.write(fd,b'v' + b'nscqvh')
   wait_view(fd,t,lambda s:'READY-alpha' in s and 'Control' in s)
   expected=b'nscqvh';until(lambda:received(h,a)==expected)
   payload=b'\r\x03\x04\x1b[A\x1b[B'+ '中文🙂'.encode()+b'\x1b[200~\x1b[24~\x02d\x02\x1b[B\x1b[201~'
   os.write(fd,payload);expected+=payload;until(lambda:received(h,a)==expected)
   os.write(fd,b'\x1b');expected+=b'\x1b';until(lambda:received(h,a)==expected)
   os.write(fd,b'\x02\x1b[Bbeta-input')
   wait_view(fd,t,lambda s:'READY-beta' in s and 'Control' in s)
   until(lambda:received(h,b)==b'beta-input');assert received(h,a)==expected
   os.write(fd,b'\x02\x1b[Aalpha-input');wait_view(fd,t,lambda s:'READY-alpha' in s)
   expected+=b'alpha-input';until(lambda:received(h,a)==expected)
   a_size=latest_size(h,a);assert a_size==(17,37),a_size
   os.write(fd,b'\x1b[13;5u');wait_view(fd,t,lambda s:'alpha [' in s and 'Control' not in s)
   os.write(fd,b'fullscreen');expected+=b'fullscreen';until(lambda:received(h,a)==expected)
   os.write(fd,b'\x1b[24~');wait_view(fd,t,lambda s:'READY-alpha' in s and 'Control' in s)
   for byte in b'\x1b[24~':os.write(fd,bytes([byte]));time.sleep(.005)
   wait_view(fd,t,lambda s:'Memory' in s and 'Control' not in s)
   os.write(fd,b'v');wait_view(fd,t,lambda s:'Control' in s and 'READY-alpha' in s)
   os.write(fd,b'\x02\x04');wait_view(fd,t,lambda s:'Memory' in s and 'Control' not in s)
   assert received(h,a)==expected
   os.write(fd,b'q');assert h.join(pid)==0
   for ref,original in before.items():assert h.show(ref)['runtime']==original
   if shutil.which('vim'):
    file=pathlib.Path(root)/'pane.txt';ref=h.create('vim-control',shutil.which('vim'),'-u','NONE','-n',str(file))
    pid,fd=h.spawn_tty();t=Terminal();wait_view(fd,t,lambda s:'vim-control' in s and 'Memory' in s)
    os.write(fd,b'\x1b[B\x1b[Bv');wait_view(fd,t,lambda s:'vim-control' in s and 'Control' in s)
    os.write(fd,b'iHello '+ '世界'.encode()+b'\x1b:w\r')
    until(lambda:file.exists() and file.read_text().strip()=='Hello 世界')
    os.write(fd,b'\x1b[24~');wait_view(fd,t,lambda s:'Memory' in s)
    os.write(fd,b'q');assert h.join(pid)==0
   print('PASS immediate v control/typeahead, literal list keys, Ctrl+C/D, arrows, paste/Unicode/Esc, prefix switch/Ctrl+Enter fullscreen, split F12 and compatible detach; session PIDs preserved; available Vim edit')
  finally:h.cleanup()
if __name__=='__main__':main()
