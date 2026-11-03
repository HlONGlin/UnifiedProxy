"""Validate P5 evidence and produce a compact, reproducible acceptance summary."""
from pathlib import Path
import hashlib
import json
from PIL import Image

assert Path.cwd() == Path('/workspace') and Path('/.dockerenv').exists()
root = Path('dist/design-audit')
documents = [json.loads((root / 'p5' / suffix / 'browser-results.json').read_text())
             for suffix in ('', 'fleet', 'operations')]
expected = {f't{i}-{mode}' for i in range(5) for mode in ('light', 'dark')}
for document in documents:
    assert not document['errors']
    assert all(check['passed'] for check in document['checks'])
    assert all(not row['failures'] for row in document['contrasts'])
for document in documents[:2]:
    assert {row['theme'] for row in document['discipline']} == expected
contrasts = [row for d in documents[:2] for row in d['contrasts']]
discipline = [row for d in documents[:2] for row in d['discipline']]


def fluorescent_fraction(document, row):
    matches = [Path(name) for name in document['screenshots']
               if Path(name).parent.name == row['theme'] and Path(name).stem == row['surface']]
    assert len(matches) == 1, (row['theme'], row['surface'])
    with Image.open(matches[0]) as screenshot:
        rgb = screenshot.convert('RGB')
        target = (194, 239, 78)  # Supplied T2 fluorescent #c2ef4e, not violet CTA.
        count = sum(n for n, color in rgb.getcolors(rgb.width * rgb.height)
                    if max(abs(color[i] - target[i]) for i in range(3)) <= 5)
        return round(count / (rgb.width * rgb.height), 6)


# Re-measure all saved T2 screenshots, including runs produced before the
# separate fluorescence field existed. Do not reinterpret light violet as lime.
fluorescence = {theme: [fluorescent_fraction(d, row) for d in documents[:2]
                       for row in d['discipline'] if row['theme'] == theme]
                for theme in sorted(expected) if theme.startswith('t2')}
themes = {}
for theme in sorted(expected):
    measurements = [m for r in contrasts if r['theme'] == theme for m in r['measurements']]
    colors = [r for r in discipline if r['theme'] == theme]
    themes[theme] = {
        'textSamples': len(measurements),
        'normalMin': min(m['ratio'] for m in measurements if m['required'] == 4.5),
        'largeMin': min(m['ratio'] for m in measurements if m['required'] == 3),
        'accentMaxFraction': max(r['accent_pixel_share'] for r in colors),
        'highlightMaxFraction': max(r['highlight_pixel_share'] for r in colors),
        'fluorescentMaxFraction': max(fluorescence.get(theme, [0])),
    }
    assert themes[theme]['normalMin'] >= 4.5 and themes[theme]['largeMin'] >= 3
    if theme.startswith('t2'):
        assert themes[theme]['fluorescentMaxFraction'] < .02
invariants = ('chart-line', 'semantic-success', 'latency-good', 'row-height',
              'rounded-md', 'duration-fast', 'font-mono')
for mode in ('light', 'dark'):
    for key in invariants:
        assert len({r['tokens'][key] for r in discipline if r['theme'].endswith(mode)}) == 1, key
events = [json.loads(line) for line in (root / 'p5-tests.json').read_text().splitlines()]
assert not any(e['Action'] == 'fail' for e in events)
assert any(e['Action'] == 'pass' and e.get('Test') == 'TestLegacyMihomoTransportFromOldestBackup' for e in events)
packages = [e['Package'] for e in events if e['Action'] == 'pass' and 'Test' not in e]
assert len(packages) == 4
live = json.loads((root / 'p5-live/results.json').read_text())
assert live['status'] == 'passed' and len(live['checks']) == 12
assert live['controllerOutageAndAgentRestart'] == 'passed'
assert '18 native interaction checks PASS' in (root / 'p5-native.log').read_text()
assert 'go build ./...: PASS' in (root / 'p5-final-check.log').read_text()
assert 'go vet ./...: PASS' in (root / 'p5-final-check.log').read_text()
summary = {
    'environment': 'orca-linux-unifiedproxy /workspace, Linux only',
    'browser': [dict(checks=len(d['checks']), screenshots=len(d['screenshots'])) for d in documents],
    'themes': themes,
    'sharedTokens': list(invariants),
    'go': dict(run=sum(e['Action'] == 'run' for e in events),
               passed=sum(e['Action'] == 'pass' and 'Test' in e for e in events),
               skipped=[e['Test'] for e in events if e['Action'] == 'skip' and 'Test' in e],
               packages=packages),
    'liveDataPlane': dict(checks=len(live['checks']), offlineRestart=live['controllerOutageAndAgentRestart']),
    'evidenceSHA256': {str(p.relative_to(root)): hashlib.sha256(p.read_bytes()).hexdigest()
                       for p in [root / 'p5-tests.json', root / 'p5-live/results.json',
                                 *(root / 'p5' / s / 'browser-results.json' for s in ('', 'fleet', 'operations'))]},
}
(root / 'p5/summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
print(json.dumps(summary, ensure_ascii=False, indent=2))
