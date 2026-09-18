#!/usr/bin/env python3
"""Profile fresh Go frontend processes with the HTTP playground fixture."""
import argparse
import json
import os
from pathlib import Path
import subprocess

from probe import ROOT

PROGRAM = r'''package main
import (
 "encoding/json"
 "os"
 "runtime/pprof"
 "time"
 "github.com/eddyw/godart/internal/driver"
)
func main() {
 profile,err:=os.Create(os.Args[3]);if err!=nil {panic(err)}
 if err:=pprof.StartCPUProfile(profile);err!=nil {panic(err)}
 phases:=map[string]float64{}
 start:=time.Now()
 stats,err:=driver.BuildWithStats(driver.Options{Dir:os.Args[1],Patterns:[]string{"."},Out:os.Args[2],Trace:func(name string,duration time.Duration){phases[name]=duration.Seconds()}})
 elapsed:=time.Since(start).Seconds()
 pprof.StopCPUProfile();profile.Close()
 if err!=nil {panic(err)}
 json.NewEncoder(os.Stdout).Encode(map[string]any{"seconds":elapsed,"phases":phases,"compiled":len(stats.Compiled),"cached":len(stats.Cached)})
}
'''


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    destination = args.output.resolve()
    if not destination.is_relative_to(ROOT / 'build'):
        parser.error('place output under build/')
    destination.mkdir(parents=True, exist_ok=False)
    (destination / 'profile.go').write_text(PROGRAM)
    binary = destination / 'profile'
    env = dict(os.environ, GOMAXPROCS='4', GOFLAGS='-p=2')
    subprocess.run(['go', 'build', '-o', str(binary), str(destination / 'profile.go')], cwd=ROOT, env=env, check=True)
    source = destination / 'source'
    source.mkdir()
    fixture = ROOT / 'experiments/resident/playground'
    original = (fixture / 'main.go').read_text()
    (source / 'go.mod').write_bytes((fixture / 'go.mod').read_bytes())
    rows = []
    profiles = []
    for index, state in enumerate(['cold', 'warm', 'warm', 'warm', 'edit', 'edit', 'edit']):
        if state != 'warm':
            (source / 'main.go').write_text(original.replace('greet("world")', f'greet("{index:05d}")'))
        profile = destination / f'{index}-{state}.pprof'
        result = subprocess.check_output([str(binary), str(source), str(destination / 'output'), str(profile)],
                                         cwd=ROOT, env=env, text=True)
        row = dict(state=state, **json.loads(result))
        rows.append(row)
        if state != 'cold':
            profiles.append(str(profile))
        print(json.dumps(row), flush=True)
    for name, flags in [('top', ['-nodecount=20']), ('cumulative', ['-cum', '-nodecount=45']),
                        ('project', ['-cum', '-nodecount=35', '-focus=github.com/eddyw/godart'])]:
        text = subprocess.check_output(['go', 'tool', 'pprof', '-top', *flags, str(binary), *profiles], cwd=ROOT, env=env, text=True)
        (destination / (name + '.txt')).write_text(text)
    (destination / 'results.json').write_text(json.dumps(dict(rows=rows), indent=2)+'\n')


if __name__ == '__main__':
    main()
