#!/usr/bin/env python3
"""Record an actual isolated TUI lifecycle to asciinema v2 and a GIF (Pillow)."""
import argparse
import json
import fcntl
import signal
import shutil
import struct
import subprocess
import sys
import termios
import os
import pathlib
import time
import tempfile
from PIL import Image, ImageDraw, ImageFont, ImageColor
from integration import Harness, ENV, drain
from screen_regression import Terminal
from ui_regression import wait_view

ROOT = pathlib.Path(__file__).resolve().parent.parent
PALETTE = dict(black='#202734', red='#ef7772', green='#9bd391', brown='#e5c07b', yellow='#e5c07b', blue='#73b4e6', magenta='#c59be3', cyan='#7dd7d0', white='#e5edf7', brightblack='#7b8798', brightred='#ff9690', brightgreen='#b2e6a7', brightyellow='#f7d794', brightblue='#9acbf0', brightmagenta='#ddb3f0', brightcyan='#a0ebe4', brightwhite='#ffffff')


def color(value, default):
    if value == 'default': return default
    if len(value) == 6 and all(c in '0123456789abcdefABCDEF' for c in value): return '#' + value
    return PALETTE.get(value, default)


def image(terminal, caption, font, bold, native_colors):
    cw, ch = round(font.getlength('M')), 23
    out = Image.new('RGB', (terminal.columns*cw+32, terminal.rows*ch+64), '#161d28')
    draw = ImageDraw.Draw(out)
    draw.text((16, 10), caption, font=font, fill='#909bae')
    cells = []
    for y in range(terminal.rows):
        for x in range(terminal.columns):
            cell = terminal.current.buffer[y][x]
            fg, bg = color(cell.fg, '#e5edf7'), color(cell.bg, '#161d28')
            native_colors.update((fg, bg))
            if cell.reverse: fg, bg = bg, fg
            left, top = 16+x*cw, 48+y*ch
            draw.rectangle((left, top, left+cw-1, top+ch-1), fill=bg)
            cells.append((cell, left, top, fg))
    for cell, left, top, fg in cells:
        if cell.data and not getattr(cell, 'hidden', False):
            draw.text((left, top), cell.data, font=bold if cell.bold else font, fill=fg)
    return out


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', default='bin/mtask')
    parser.add_argument('--font', default='/System/Library/Fonts/Menlo.ttc')
    args = parser.parse_args()
    font = ImageFont.truetype(args.font, 16)
    bold = ImageFont.truetype(args.font, 16, index=1) if args.font.endswith('.ttc') else font
    destination = ROOT/'docs/assets'; destination.mkdir(parents=True, exist_ok=True)
    previous_env = dict(ENV)
    # The recorder's VT model supports true color. Do not inherit Codex's
    # NO_COLOR setting; preserve the application's actual SGR colors.
    ENV.pop('NO_COLOR', None)
    ENV['COLORTERM'] = 'truecolor'
    ENV['CLICOLOR'] = '1'
    columns, rows = 100, 24
    frames, durations, events, chapters = [], [], [], []
    native_colors = {'#909bae', '#161d28'}
    with tempfile.TemporaryDirectory(prefix='mtask-demo-') as root:
        ENV['HOME'] = root
        ENV.pop('ENV', None)
        h = Harness(root, pathlib.Path(args.binary).resolve())
        try:
            h.cli('settings','set','language','en')
            h.cli('settings','set','history-retention','off')
            project = pathlib.Path(root)/'mtask'; project.mkdir()
            for name in ['cmd', 'docs', 'internal', 'scripts', 'examples']:
                (project/name).mkdir()
            for name in ['README.md', 'go.mod', 'Makefile']:
                (project/name).write_bytes((ROOT/name).read_bytes())
            (project/'events.log').write_text('Watching project files.\n')
            subprocess.run(['git', 'init', '-q', str(project)], check=True)
            def create(name, *command):
                record=json.loads(h.cli('new','-d','--json','--name',name,'--cwd',str(project),'--',*command).stdout)
                return record['spec']['id']
            dev=create('dev-shell','/usr/bin/env','PS1=mtask$ ','/bin/sh','-i')
            create('api-server',sys.executable,'-u','-m','http.server','0','--bind','127.0.0.1')
            create('docs-server',sys.executable,'-u','-m','http.server','0','--bind','127.0.0.1')
            watcher="import pathlib,time; p=pathlib.Path('.'); print('Watching cmd/, internal/, docs/',flush=True)\nwhile True:\n print('Files: '+str(sum(x.is_file() for x in p.rglob('*'))),flush=True);time.sleep(2)"
            create('file-watch',sys.executable,'-u','-c',watcher)
            create('log-tail','/usr/bin/tail','-f',str(project/'events.log'))
            for name in ['build-shell', 'release-shell']:
                create(name,'/usr/bin/env','PS1='+name+'$ ','/bin/sh','-i')
            htop=shutil.which('htop')
            last_session='system-monitor' if htop else 'scratch-shell'
            if htop:
                pids=','.join(str(r['runtime']['child_pid']) for r in json.loads(h.cli('ls','--json').stdout))
                create(last_session,htop,'--readonly','--pid',pids,'--delay','10')
            else:
                create(last_session,'/usr/bin/env','PS1=scratch-shell$ ','/bin/sh','-i')
            for name,command in [
                ('git-status',['git','status','--short']),
                ('readme-lines',['/usr/bin/wc','-l','README.md']),
                ('directory-list',['/bin/ls']),
            ]:
                h.finished(create(name,*command))
            before={r['spec']['id']:r['runtime'] for r in json.loads(h.cli('ls','--json').stdout) if r['runtime']['state']=='running'}
            assert len(before)==8
            terminal = Terminal(columns, rows); start=time.monotonic(); original=terminal.feed
            def feed(data):
                if data:
                    events.append([round(time.monotonic()-start,3),'o',data.decode('utf-8','replace')])
                    original(data)
            terminal.feed=feed
            pid,fd=h.spawn_tty()
            fcntl.ioctl(fd,termios.TIOCSWINSZ,struct.pack('HHHH',rows,columns,0,0))
            os.kill(pid,signal.SIGWINCH)
            def scene(caption, seconds=2.5):
                chapters.append({'caption':caption,'frame':len(frames)})
                for _ in range(round(seconds*4)):
                    tick=time.monotonic()
                    terminal.feed(drain(fd,.20))
                    frames.append(image(terminal,caption,font,bold,native_colors));durations.append(250)
                    time.sleep(max(0,.25-(time.monotonic()-tick)))
            wait_view(fd,terminal,lambda text:last_session in text and 'Running 8' in text and 'Stop 3' in text and '0.0%' in text)
            colors={terminal.current.buffer[y][x].fg for y in range(rows) for x in range(columns)}
            assert '70e1c8' in colors, 'the application did not emit its normal accent color'
            scene('Task list · 8 running sessions',3)
            os.write(fd,b'\r');wait_view(fd,terminal,lambda text:not terminal.alternate and 'mtask$' in text)
            ls = 'ls -G' if sys.platform=='darwin' else 'ls --color=auto'
            os.write(fd,(ls+'\r').encode())
            wait_view(fd,terminal,lambda text:'examples' in text and 'README.md' in text)
            scene('Enter · open dev-shell',3)
            os.write(fd,b'\x1b[24~');wait_view(fd,terminal,lambda text:terminal.alternate and 'Running 8' in text and last_session in text)
            scene('F12 · return to the task list',2)
            if htop:
                os.write(fd,b'\x1b[B'*7+b'\r')
                wait_view(fd,terminal,lambda text:'PID' in text and 'Command' in text and 'Running 8' not in text)
                scene('Enter · run htop in its own terminal',3)
                os.write(fd,b'\x1b[24~');wait_view(fd,terminal,lambda text:'Running 8' in text and 'Memory' in text)
                os.write(fd,b'\x1b[A'*7)
            os.write(fd,b'v');wait_view(fd,terminal,lambda text:'README.md' in text and 'Control' in text)
            os.write(fd,b"printf 'Input works inside the monitor\\n'\r")
            wait_view(fd,terminal,lambda text:'Input works inside' in text)
            scene('v · type directly in the monitor',3)
            os.write(fd,b'\x1b[B'*3);wait_view(fd,terminal,lambda text:'Files:' in text)
            scene('Up / Down · switch the monitor',2.5)
            os.write(fd,b'\x1b[24~');wait_view(fd,terminal,lambda text:'Control' not in text and 'Memory' in text)
            os.write(fd,b'c');wait_view(fd,terminal,lambda text:'Connection modes' in text)
            scene('c · choose how to connect',2.5)
            os.write(fd,b'\x1b');wait_view(fd,terminal,lambda text:'Connection modes' not in text)
            os.write(fd,b'\x1b[A'*3+b'\r');wait_view(fd,terminal,lambda text:not terminal.alternate and 'dev-shell [' in text)
            os.write(fd,b"printf 'Shell PID: %s\\n' \"$$\"\r")
            wait_view(fd,terminal,lambda text:('Shell PID: '+str(before[dev]['child_pid'])) in text)
            scene('Enter · reconnect to the same process',2.5)
            os.write(fd,b'\x02d');wait_view(fd,terminal,lambda text:terminal.alternate and 'Running 8' in text and 'Memory' in text)
            scene('Ctrl+B then d · return to the list',2)
            os.write(fd,b's');wait_view(fd,terminal,lambda text:'Stop this session' in text)
            scene('s · review the stop confirmation',2.5)
            os.write(fd,b'y');h.finished(dev);os.write(fd,b'\t')
            wait_view(fd,terminal,lambda text:'Stop 4' in text and 'Duration' in text and 'dev-shell' in text)
            os.write(fd,b'\x1b[A'*4)
            scene('Stop · finished sessions and retained output',2)
            os.write(fd,b'\r');wait_view(fd,terminal,lambda text:'Output history' in text and 'Shell PID:' in text)
            scene('Enter · inspect session history',2.5)
            os.write(fd,b'\x1b');wait_view(fd,terminal,lambda text:'Output history' not in text and 'Duration' in text)
            os.write(fd,b'\x1b[Z');wait_view(fd,terminal,lambda text:'Running 7' in text and 'Memory' in text)
            scene('Running · the other sessions keep working',3)
            for ref,runtime in before.items():
                if ref!=dev:
                    after=h.show(ref)
                    assert after['health']=='online'
                    assert after['runtime']['child_pid']==runtime['child_pid']
            os.write(fd,b'q');assert h.join(pid)==0
        finally:
            h.cleanup()
            ENV.clear(); ENV.update(previous_env)
    # One shared palette avoids color flicker between frames. Sample every
    # chapter so shell colors and confirmation colors are represented as well.
    samples=[frames[c['frame']].resize((516,308)) for c in chapters]
    sheet=Image.new('RGB',(516*4,308*((len(samples)+3)//4)))
    for i,sample in enumerate(samples):sheet.paste(sample,((i%4)*516,(i//4)*308))
    adaptive=sheet.quantize(colors=256,method=Image.Quantize.MEDIANCUT).getpalette()
    colors=[ImageColor.getrgb(value) for value in sorted(native_colors)]
    assert len(colors)<256
    for offset in range(0,len(adaptive),3):
        rgb=tuple(adaptive[offset:offset+3])
        if rgb not in colors:colors.append(rgb)
    palette=Image.new('P',(1,1))
    palette.putpalette([channel for rgb in colors[:256] for channel in rgb])
    gif=[frame.quantize(palette=palette,dither=Image.Dither.NONE) for frame in frames]
    gif[0].save(destination/'demo.gif',save_all=True,append_images=gif[1:],duration=durations,loop=0,optimize=True)
    header={'version':2,'width':columns,'height':rows,'title':'mtask: multiple terminal sessions','env':{'TERM':'xterm-256color','COLORTERM':'truecolor'}}
    (destination/'demo.cast').write_text('\n'.join(json.dumps(x,ensure_ascii=False) for x in [header,*events])+'\n')
    preview=pathlib.Path(tempfile.gettempdir())/'mtask-demo-preview.png'
    contact=Image.new('RGB',(1032*2,616*3),'#161d28')
    selections=[0,3,4,6,9,11] if htop else [0,1,3,5,8,10]
    for i,index in enumerate(selections):contact.paste(frames[chapters[index]['frame']],((i%2)*1032,(i//2)*616))
    contact.save(preview)
    print('PASS recorded 8 real running sessions and 3 historical tasks in English; native terminal colors preserved')
    print('Preview: '+str(preview))


if __name__=='__main__':
    main()
