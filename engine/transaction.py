"""Hash-checked local transaction and V4 rollback manifest compatibility."""
from pathlib import Path, PureWindowsPath
from dataclasses import dataclass
import hashlib,os,tempfile,json,stat,datetime
LATEST='ORFixRepair_latest_rollback.json'
def sha(b):return hashlib.sha256(b).hexdigest()
def stamp():return datetime.datetime.now().strftime('%Y-%m-%d_%H-%M-%S_%f')
def safe(root,rel):
    w=PureWindowsPath(rel);p=Path(rel.replace('\\','/'))
    if w.drive or w.root or p.is_absolute() or '..' in p.parts or not p.parts:raise ValueError('不安全路径 '+rel)
    p=root/p
    if p.is_symlink() or not p.resolve().is_relative_to(root.resolve()):raise ValueError('路径越界 '+rel)
    return p

def rel(root,p):return p.relative_to(root).as_posix()
def atomic(path,data):
    fd,tmp=tempfile.mkstemp(prefix='.orfix-',suffix='.tmp',dir=path.parent)
    try:
        with os.fdopen(fd,'wb') as f:f.write(data);f.flush();os.fsync(f.fileno())
        if path.exists():os.chmod(tmp,stat.S_IMODE(path.stat().st_mode))
        os.replace(tmp,path)
    finally:
        if os.path.exists(tmp):os.unlink(tmp)
def exclusive(path,data):
    with path.open('xb') as f:f.write(data);f.flush();os.fsync(f.fileno())

def precheck(root,edit):
    p,a,b=edit
    if safe(root,rel(root,p))!=p or p.is_symlink() or not p.is_file() or p.read_bytes()!=a:raise ValueError('扫描后文件变更或路径不安全：'+str(p))
    if not p.stat().st_mode & stat.S_IWUSR:raise PermissionError('只读文件 '+str(p))

def commit(root,edits,when):
    if len({p.resolve() for p,a,b in edits})!=len(edits):raise ValueError('重复修改目标')
    for e in edits:precheck(root,e)
    backups=[];written=[]
    try:
        for p,a,b in edits:
            backup=p.with_name(p.name+'_'+when+'.bak');exclusive(backup,a);backups.append(backup)
        for e in edits:
            precheck(root,e);p,a,b=e;atomic(p,b);written.append(e)
    except Exception:
        for p,a,b in reversed(written):
            if p.read_bytes()==b:atomic(p,a)
        raise
    return [{'ini':rel(root,p),'backup':rel(root,bak),'original_sha256':sha(a),'patched_sha256':sha(b)} for (p,a,b),bak in zip(edits,backups)]

def merge(old,new):
    result={e['ini']:dict(e) for e in old}
    for e in new:
        if e['ini'] in result:
            prev=result[e['ini']]
            if prev['patched_sha256']!=e['original_sha256']:raise ValueError('跨阶段 Hash 不连续：'+e['ini'])
            prev['patched_sha256']=e['patched_sha256']
        else:result[e['ini']]=dict(e)
    return list(result.values())

def manifest(root,entries,when):
    if not entries:return
    raw=json.dumps({'format':1,'created':datetime.datetime.now().isoformat(),'items':entries},ensure_ascii=False,indent=2).encode()
    exclusive(root/f'ORFixRepair_{when}_rollback.json',raw);atomic(root/LATEST,raw)

def restore(root):
    mp=safe(root,LATEST);m=json.loads(mp.read_bytes());
    if m.get('format')!=1 or not isinstance(m.get('items'),list):raise ValueError('回滚清单格式无效')
    when=stamp();logs=[];edits=[]
    for e in m['items']:
        p=safe(root,e['ini']);b=safe(root,e['backup']);curr=p.read_bytes();old=b.read_bytes()
        if sha(curr)!=e['patched_sha256'] or sha(old)!=e['original_sha256']:raise ValueError('回滚冲突，整批保留：'+e['ini'])
        edits.append((p,curr,old))
    restored=commit(root,edits,when)
    try:os.replace(mp,root/f'ORFixRepair_{when}_已回滚.json')
    except Exception:
        for p,a,b in reversed(edits):
            if p.read_bytes()==b:atomic(p,a)
        raise
    return len(restored),['已恢复：'+e['ini'] for e in restored]
