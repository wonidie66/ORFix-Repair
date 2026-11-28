"""Compile the optional desktop glass shader using the installed Qt toolchain.
No network access; no user or Mod files are touched. Frozen apps use bundled QSB.
"""
from __future__ import annotations
import argparse
import os
from pathlib import Path
import shutil
import subprocess
import sys
import sysconfig
import tempfile

BASE = Path(__file__).resolve().parent
SOURCE = BASE / 'shaders' / 'liquidglass.frag'
OUTPUT = BASE / 'shaders' / 'liquidglass.frag.qsb'


def compiler_candidates():
    import PySide6
    root = Path(PySide6.__file__).resolve().parent
    scripts = Path(sysconfig.get_path('scripts'))
    paths = [root / 'qsb.exe', root / 'qsb', root / 'Qt/bin/qsb.exe', root / 'Qt/bin/qsb',
             scripts / 'pyside6-qsb.exe', scripts / 'pyside6-qsb']
    for name in ('pyside6-qsb', 'qsb'):
        found = shutil.which(name)
        if found:
            paths.append(Path(found))
    # Explicit override for a locally installed Qt SDK.
    if os.environ.get('ORFIX_QSB'):
        paths.insert(0, Path(os.environ['ORFIX_QSB']))
    seen = set()
    for path in paths:
        if path.is_file() and str(path) not in seen:
            seen.add(str(path))
            yield path


def prepare_shader(force=False):
    if getattr(sys, 'frozen', False):
        return OUTPUT if OUTPUT.is_file() and OUTPUT.stat().st_size > 0 else None
    if not force and OUTPUT.is_file() and OUTPUT.stat().st_size > 0 and OUTPUT.stat().st_mtime_ns >= SOURCE.stat().st_mtime_ns:
        return OUTPUT
    errors = []
    for compiler in compiler_candidates():
        tmp = None
        try:
            fd, name = tempfile.mkstemp(suffix='.qsb', dir=SOURCE.parent)
            os.close(fd)
            tmp = Path(name)
            result = subprocess.run([str(compiler), '--glsl', '100 es,120,150', '--hlsl', '50', '--msl', '12', '-o', str(tmp), str(SOURCE)],
                                    capture_output=True, text=True, errors='replace', timeout=45,
                                    creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
            if result.returncode == 0 and tmp.stat().st_size > 0:
                os.replace(tmp, OUTPUT)
                return OUTPUT
            errors.append(f'{compiler.name}: {result.stderr or result.stdout}')
        except (OSError, subprocess.SubprocessError) as exc:
            errors.append(f'{compiler.name}: {exc}')
        finally:
            if tmp is not None:
                tmp.unlink(missing_ok=True)
    print('Liquid Glass shader unavailable; using the native silver fallback. ' + ('; '.join(errors) if errors else 'Qt qsb tool not found.'), file=sys.stderr)
    return None


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--required', action='store_true')
    args = parser.parse_args()
    shader = prepare_shader(force=True)
    if shader:
        print(f'Compiled: {shader}')
    sys.exit(1 if args.required and shader is None else 0)
