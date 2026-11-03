// Compatibility entrypoint: equivalent native behavior is verified in Linux.
const {spawnSync}=require('node:child_process');
if(process.platform!=='linux'||process.cwd()!=='/workspace')throw Error('Run in orca-linux /workspace');
const result=spawnSync('/root/.venvs/unifiedproxy/bin/python',['scripts/acceptance/native-controls.py'],{stdio:'inherit'});
process.exit(result.status??1);
