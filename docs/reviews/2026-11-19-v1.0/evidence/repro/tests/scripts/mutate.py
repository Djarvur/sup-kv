#!/usr/bin/env python3
"""Mutation-probe harness for the sup-kv test audit (scratch copy only).

For every mutation: apply ONE exact-string edit to one production file, check
it compiles, run the whole suite on darwin (host) and on linux/arm64
(container) without -race, rerun failing packages once to weed out flakes,
record the killing tests, restore the file byte for byte.
"""
import json
import os
import re
import subprocess
import sys
import time

W = '<scratch>/work-tests'
SRC = W + '/sup-kv'
LX = W + '/scripts/lx.sh'
LOG = W + '/out/mutations.jsonl'
RAW = W + '/out/mutations_raw'
os.makedirs(RAW, exist_ok=True)

ENV = dict(os.environ, SUPKV_SIM_SEED='1')

sys.path.insert(0, W + '/scripts')
from mutlist import MUTATIONS  # noqa: E402


def sh(cmd, timeout=900, env=ENV, cwd=SRC):
    try:
        p = subprocess.run(cmd, shell=True, cwd=cwd, env=env, capture_output=True, text=True, timeout=timeout)
        return p.returncode, p.stdout + p.stderr
    except subprocess.TimeoutExpired as e:
        return 124, (e.stdout or '') + (e.stderr or '') + '\nHARNESS TIMEOUT'


def parse(out):
    """Return (failed tests, failed packages, timed-out markers)."""
    tests = sorted(set(re.findall(r'^\s*--- FAIL: (\S+)', out, re.M)))
    pkgs = sorted(set(re.findall(r'^FAIL\s+(github\.com/\S+)', out, re.M)))
    panics = re.findall(r'panic: test timed out after \S+\n\s*running tests:\n((?:\s+\S+ \(\S+\)\n)+)', out)
    timed = sorted(set(t.strip().split(' ')[0] for blk in panics for t in blk.strip().split('\n')))
    other_panic = sorted(set(re.findall(r'^panic: (?!test timed out)(.{0,120})', out, re.M)))
    build = sorted(set(re.findall(r'^(\S+\.go:\d+:\d+: .*)$', out, re.M)))[:5] if '[build failed]' in out or 'build failed' in out else []
    return tests, pkgs, timed, other_panic, build


def run_suite(platform, pkgs='./...'):
    if platform == 'darwin':
        return sh(f'go test -count=1 -p 1 -timeout 150s {pkgs} 2>&1')
    return sh(f"{LX} 'go test -count=1 -p 1 -timeout 150s {pkgs} 2>&1'")


def rel(pkgs):
    return ' '.join('./' + p.replace('github.com/Djarvur/sup-kv/', '') + '/' for p in pkgs)


def judge(platform, mid):
    rc, out = run_suite(platform)
    open(f'{RAW}/{mid}_{platform}_1.txt', 'w').write(out)
    tests, pkgs, timed, panics, build = parse(out)
    if build:
        return {'status': 'BUILD_FAILED', 'detail': build}
    if rc == 0:
        return {'status': 'SURVIVED', 'tests': []}
    # confirm: rerun only the failing packages; a kill must reproduce
    rc2, out2 = run_suite(platform, rel(pkgs) if pkgs else './...')
    open(f'{RAW}/{mid}_{platform}_2.txt', 'w').write(out2)
    tests2, pkgs2, timed2, panics2, _ = parse(out2)
    both = sorted((set(tests) | set(timed)) & (set(tests2) | set(timed2)))
    res = {'tests': both, 'first_run_only': sorted((set(tests) | set(timed)) - set(both)),
           'timeouts': sorted(set(timed) & set(timed2)), 'panics': sorted(set(panics) & set(panics2))}
    if both or (panics and panics2) or (rc2 != 0 and set(pkgs) & set(pkgs2) and not tests and not tests2):
        res['status'] = 'KILLED'
        res['pkgs'] = sorted(set(pkgs) & set(pkgs2))
    else:
        res['status'] = 'SURVIVED_FLAKE_ONLY'
    return res


def apply(m):
    path = os.path.join(SRC, m['file'])
    orig = open(path).read()
    start = 0
    if m.get('after'):
        start = orig.find(m['after'])
        if start < 0:
            return orig, None, 'anchor not found'
    cnt = orig.count(m['old'], start) if m.get('after') else orig.count(m['old'])
    if m.get('after'):
        idx = orig.find(m['old'], start)
        if idx < 0:
            return orig, None, 'old not found after anchor'
    else:
        if cnt != 1:
            return orig, None, f'old occurs {cnt} times'
        idx = orig.find(m['old'])
    mutated = orig[:idx] + m['new'] + orig[idx + len(m['old']):]
    line = orig.count('\n', 0, idx) + 1
    return orig, (mutated, line), None


def main():
    only = set(sys.argv[2:]) if len(sys.argv) > 2 else None
    dry = len(sys.argv) > 1 and sys.argv[1] == 'dry'
    done = set()
    if os.path.exists(LOG) and not dry:
        for ln in open(LOG):
            done.add(json.loads(ln)['id'])
    for m in MUTATIONS:
        if only and m['id'] not in only:
            continue
        if m['id'] in done and not only:
            continue
        path = os.path.join(SRC, m['file'])
        orig, res, err = apply(m)
        if err:
            print(m['id'], 'INVALID:', err, flush=True)
            if not dry:
                open(LOG, 'a').write(json.dumps({'id': m['id'], 'file': m['file'], 'desc': m['desc'], 'error': err}) + '\n')
            continue
        mutated, line = res
        if dry:
            open(path, 'w').write(mutated)
            try:
                rc, out = sh('go build ./... 2>&1 && go vet ./' + os.path.dirname(m['file']) + '/ 2>&1 | head -5')
            finally:
                open(path, 'w').write(orig)
            print(m['id'], f"{m['file']}:{line}", 'compiles' if rc == 0 and 'cannot' not in out else 'COMPILE/VET: ' + out.strip()[:300], flush=True)
            continue
        t0 = time.time()
        rec = {'id': m['id'], 'file': m['file'], 'line': line, 'desc': m['desc'], 'diff': m.get('diff', '')}
        open(path, 'w').write(mutated)
        try:
            rc, out = sh('go build ./... 2>&1')
            if rc != 0:
                rec['error'] = 'does not compile: ' + out.strip()[:400]
            else:
                rec['darwin'] = judge('darwin', m['id'])
                rec['linux'] = judge('linux', m['id'])
        finally:
            open(path, 'w').write(orig)
        rec['secs'] = round(time.time() - t0)
        open(LOG, 'a').write(json.dumps(rec) + '\n')
        d, l = rec.get('darwin', {}), rec.get('linux', {})
        print(m['id'], f"{m['file']}:{line}", 'darwin=' + d.get('status', '?'), ','.join(d.get('tests', [])[:4]),
              '| linux=' + l.get('status', '?'), ','.join(l.get('tests', [])[:4]), f"({rec['secs']}s)", rec.get('error', ''), flush=True)
    rc, out = sh(f'diff -rq --exclude=.git --exclude=bin <repo> {SRC}')
    print('TREE PRISTINE' if rc == 0 else 'TREE DIFFERS:\n' + out, flush=True)


if __name__ == '__main__':
    main()
