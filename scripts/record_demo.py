#!/usr/bin/env python3
"""Record an actual isolated TUI lifecycle to asciinema v2 and a GIF (Pillow)."""
import argparse
import json
import os
import pathlib
import time
import tempfile
from PIL import Image, ImageDraw, ImageFont
from integration import Harness, drain
from screen_regression import Terminal
from ui_regression import wait_view

ROOT = pathlib.Path(__file__).resolve().parent.parent
PALETTE = dict(black='#202734', red='#ef7772', green='#9bd391', brown='#e5c07b', yellow='#e5c07b', blue='#73b4e6', magenta='#c59be3', cyan='#7dd7d0', white='#e5edf7', brightblack='#7b8798', brightred='#ff9690', brightgreen='#b2e6a7', brightyellow='#f7d794', brightblue='#9acbf0', brightmagenta='#ddb3f0', brightcyan='#a0ebe4', brightwhite='#ffffff')


def color(value, default):
    if value == 'default': return default
    if len(value) == 6 and all(c in '0123456789abcdefABCDEF' for c in value): return '#' + value
    return PALETTE.get(value, default)


def image(terminal, caption, font):
    cw, ch = round(font.getlength('M')), 23
    out = Image.new('RGB', (terminal.columns*cw+32, terminal.rows*ch+64), '#161d28')
    draw = ImageDraw.Draw(out)
    draw.text((16, 10), caption, font=font, fill='#a8d7ff')
    for y in range(terminal.rows):
        for x in range(terminal.columns):
            cell = terminal.current.buffer[y][x]
            fg, bg = color(cell.fg, '#e5edf7'), color(cell.bg, '#161d28')
            if cell.reverse: fg, bg = bg, fg
            left, top = 16+x*cw, 48+y*ch
            draw.rectangle((left, top, left+cw, top+ch), fill=bg)
            if cell.data: draw.text((left, top), cell.data, font=font, fill=fg)
    return out


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', default='bin/mtask')
    parser.add_argument('--font', default='/System/Library/Fonts/Menlo.ttc')
    args = parser.parse_args()
    font = ImageFont.truetype(args.font, 16)
    destination = ROOT/'docs/assets'; destination.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='mtask-demo-') as root:
        h = Harness(root, pathlib.Path(args.binary).resolve())
        frames, durations, events = [], [], []
        try:
            h.cli('settings','set','language','en')
            h.cli('settings','set','resources','off')
            h.cli('settings','set','history-retention','off')
            project = pathlib.Path(root)/'project'; project.mkdir()
            h.cli('new','-d','--name','work','--cwd',str(project),'--','/usr/bin/env','PS1=work$ ','/bin/sh','-i')
            terminal = Terminal(); start=time.monotonic(); original=terminal.feed
            def feed(data):
                events.append([round(time.monotonic()-start,3),'o',data.decode('utf-8','replace')]); original(data)
            terminal.feed=feed
            pid,fd=h.spawn_tty()
            def scene(caption):
                for _ in range(12):
                    tick=time.monotonic()
                    terminal.feed(drain(fd,.20))
                    frames.append(image(terminal,caption,font));durations.append(250)
                    time.sleep(max(0,.25-(time.monotonic()-tick)))
            wait_view(fd,terminal,lambda text:'work' in text)
            scene('1 / 10   Open the task list')
            os.write(fd,b'\r');wait_view(fd,terminal,lambda text:'work$' in text)
            scene('2 / 10   Enter: attach to the running shell')
            os.write(fd,b"printf 'Task output is retained.\\nHello from the original process.\\n'\r")
            wait_view(fd,terminal,lambda text:'Hello from the original process.' in text)
            scene('3 / 10   Run commands in the original terminal')
            os.write(fd,b'\x02d');wait_view(fd,terminal,lambda text:'Running' in text and 'work' in text)
            scene('4 / 10   Ctrl+B, d: detach and keep the task')
            os.write(fd,b'v');wait_view(fd,terminal,lambda text:'work$' in text)
            scene('5 / 10   v: monitor the task without sending input')
            os.write(fd,b'v');wait_view(fd,terminal,lambda text:'Running' in text and 'work$' not in text)
            scene('6 / 10   v: return to the regular list')
            os.write(fd,b's');wait_view(fd,terminal,lambda text:'Stop this session' in text)
            scene('7 / 10   s: stop requires confirmation')
            os.write(fd,b'y');h.finished('work');os.write(fd,b'\t')
            wait_view(fd,terminal,lambda text:'work' in text and 'Duration' in text)
            scene('8 / 10   The stopped task remains in history')
            os.write(fd,b'\r');wait_view(fd,terminal,lambda text:'Output history' in text)
            scene('9 / 10   Enter: read retained output')
            os.write(fd,b'\x1b');wait_view(fd,terminal,lambda text:'Output history' not in text and 'work' in text)
            os.write(fd,b'\x03');wait_view(fd,terminal,lambda text:not terminal.alternate)
            scene('10 / 10   Ctrl+C: exit cleanly')
            assert h.join(pid)==0
        finally: h.cleanup()
    frames[0].save(destination/'demo.gif',save_all=True,append_images=frames[1:],duration=durations,loop=0,optimize=True)
    header={'version':2,'width':80,'height':24,'title':'mtask isolated shell lifecycle','env':{'TERM':'xterm-256color'}}
    (destination/'demo.cast').write_text('\n'.join(json.dumps(x,ensure_ascii=False) for x in [header,*events])+'\n')
    print('PASS recorded actual temporary-shell lifecycle: docs/assets/demo.gif and demo.cast')

if __name__=='__main__':main()
