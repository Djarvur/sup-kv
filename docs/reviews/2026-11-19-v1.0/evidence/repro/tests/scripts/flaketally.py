#!/usr/bin/env python3
"""Tally known flake signatures over the raw outputs of the mutation runs."""
import glob, os, re, collections, json
W='<scratch>/work-tests'
sig = {
  'port TOCTOU (address already in use)': re.compile(r'address already in use'),
  'SIGTERM before signal handler (signal: terminated)': re.compile(r'signal: terminated'),
  'package timeout panic': re.compile(r'panic: test timed out'),
}
runs = collections.Counter(); hits = collections.defaultdict(collections.Counter); where = collections.defaultdict(list)
for f in sorted(glob.glob(W+'/out/mutations_raw/*_1.txt')):
    plat = 'darwin' if '_darwin_' in f else 'linux'
    runs[plat] += 1
    txt = open(f).read()
    for name, rx in sig.items():
        if rx.search(txt):
            hits[name][plat] += 1
            tests = sorted(set(re.findall(r'--- FAIL: (\S+)', txt)))
            where[name].append((os.path.basename(f)[:3], plat, [t for t in tests if '/' not in t][:4]))
print('full-suite first runs:', dict(runs))
for name in sig:
    print(name, dict(hits[name]))
    for w in where[name][:14]:
        print('   ', w)
