#!/root/.venvs/unifiedproxy/bin/python
"""Opt-in real Mihomo parser using checksum-verified local geodata."""
import os,sys,shutil,hashlib
from pathlib import Path
assert Path.cwd()==Path('/workspace') and Path('/.dockerenv').exists()
args=sys.argv[1:]
if '-d' in args:
 target=Path(args[args.index('-d')+1])
 for name in ('geoip.dat','geosite.dat','geoip.metadb'):
  source=Path('/workspace/dist/design-audit/geodata')/name
  expected=source.with_name(name+'.sha256sum').read_text().split()[0]
  assert hashlib.sha256(source.read_bytes()).hexdigest()==expected, 'geodata checksum mismatch: '+name
  shutil.copyfile(source,target/name)
os.execv('/workspace/dist/design-audit/mihomo',['mihomo',*args])
