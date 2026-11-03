"""Read-only source inventory. Run in orca-linux /workspace before UI edits."""
import json
import re
from html.parser import HTMLParser
from pathlib import Path


class Inventory(HTMLParser):
    def __init__(self):
        super().__init__()
        self.ids, self.controls, self.views = [], [], []

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if attrs.get('id'):
            self.ids.append(attrs['id'])
        if tag in ('button', 'input', 'select', 'textarea', 'form', 'a', 'summary'):
            self.controls.append({'tag': tag, **attrs})
        if any(k.startswith('data-') and any(x in k for x in ('tab', 'view', 'general')) for k in attrs):
            self.views.append({'tag': tag, **attrs})


if __name__ == '__main__':
    assert Path.cwd() == Path('/workspace') and Path('/.dockerenv').exists()
    files = {}
    for path in sorted(Path('web').glob('*')):
        if path.suffix not in ('.html', '.js', '.css'):
            continue
        source = path.read_text()
        parser = Inventory()
        if path.suffix == '.html':
            parser.feed(source)
        files[str(path)] = {
            'ids': parser.ids, 'controls': parser.controls, 'tabs': parser.views,
            'templateIDs': sorted(set(re.findall(r'\bid=["\']([\w-]+)["\']', source))),
            'variables': sorted(set(re.findall(r'(--[\w-]+)\s*:', source))),
            'assets': re.findall(r'(?:src|href)="([^"{}]+)"', source),
            'apiPaths': sorted(set(re.findall(r'["\'](/api/[\w/?.-]+)', source))),
        }
    output = Path('docs/design')
    output.mkdir(parents=True, exist_ok=True)
    (output / 'p1-source-inventory.json').write_text(json.dumps(files, ensure_ascii=False, indent=2)+'\n')
    print(f'Inventoried {len(files)} files, {sum(len(f["controls"]) for f in files.values())} static controls')
