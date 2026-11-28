"""Unified repair plan, independent of Qt; no runtime Go or subprocess."""
from dataclasses import dataclass,field
from pathlib import Path
import os
from .parser import Doc,Section
from .diagnostics import Decision,write_reports
from .rules import material,complex,face,migration
from .transaction import *
@dataclass
class Plan:
    path:Path;original:bytes;patched:bytes;records:list;companions:list=field(default_factory=list)
@dataclass
class Result:
    phase:str;scanned:int=0;modified:int=0;face_risks:list=field(default_factory=list);changes:list=field(default_factory=list);errors:list=field(default_factory=list);warnings:list=field(default_factory=list);log_path:str='';records:list=field(default_factory=list);entries:list=field(default_factory=list)

def material_plan(path,data):
    d=Doc(path,data);records=[];companions=[]
    if d.unsafe_global() or d.legacy():
        return Plan(path,data,data,[Decision('SAFE-001','file','blocked',reason='命名空间/重复区块/include/ShaderRegex 或历史 Reflection，保留原样',file=str(path))])
    out,r,companions=migration.migrate(d);records+=r
    out,r,handled=migration.recipe(Doc(path,out));records+=r
    if not handled:
        for fn in (material.safe_texfx,material.basic,material.enhanced,complex.transform,lambda d:complex.transform(d,True)):
            out,r=fn(Doc(path,out));records+=r
    # Intermediate rejected groups may be resolved by a later rule. Retain
    # evidence, but mark superseded rather than presenting stale final skips.
    applied={r.section.lower() for r in records if r.decision=='auto_repair' and r.rule_id in ('MAT-004','MAT-006')}
    for r in records:
        if r.decision=='needs_diagnosis' and r.section.lower() in applied:r.decision='superseded';r.reason+='；后续规则已生成候选修改（仍需实测）'
    final_doc=Doc(path,out)
    explained={r.section.lower() for r in records}
    for sec in final_doc.sections:
        if not sec.key.startswith('textureoverride') or sec.key in explained or 'face' in sec.key:continue
        bindings=material.bound(final_doc,sec)
        if not bindings:continue
        from .diagnostics import decision
        from .parser import FIX,code,outfit,classify
        if any(FIX.fullmatch(l) for _,l in final_doc.active(sec)):
            records.append(decision(final_doc,sec,'SAFE-001','no_change',['发现已有修复调用'],'不重复添加；需确认实际绘制路径及渲染状态'))
        else:
            kind,why=classify(final_doc,bindings)
            records.append(decision(final_doc,sec,'SAFE-001','needs_diagnosis',['已发现材质绑定'],why or ('角色网格关系/绘制路径不满足自动修复合同' if not outfit(final_doc,sec) else '未满足具体规则全部前提')))
    return Plan(path,data,out,records,companions)

def face_plan(path,data):
    out,records=face.analyze(Doc(path,data));return Plan(path,data,out,records)

def files(root):
    for parent,dirs,names in os.walk(root,followlinks=False):
        dirs[:]=sorted(x for x in dirs if not Path(parent,x).is_symlink() and not x.lower().startswith('disabled'))
        for name in sorted(names):
            p=Path(parent,name)
            if p.suffix.lower()=='.ini' and not p.is_symlink() and not name.lower().startswith('disabled') and name.lower()!='orfix.ini':yield p

def scan(root,facemode=False):
    plans=[];errors=[];scanned=0
    for p in files(root):
        scanned+=1
        try:
            if p.stat().st_size>8*1024*1024:raise ValueError('INI 大于 8 MiB')
            data=p.read_bytes()
            if b'\0' in data:raise ValueError('NUL 数据')
            plans.append((face_plan if facemode else material_plan)(p,data))
        except (UnicodeError,ValueError,OSError) as e:errors.append(f'{p}: {e}')
    return plans,scanned,errors

def env_warnings(root,plans):
    content=b'\n'.join(p.patched for p in plans).lower();out=[]
    ancestors=[root,*list(root.parents)[:4]]
    if b'commandlist\\global\\orfix\\' in content and not any((p/q).is_file() for p in ancestors for q in ('ORFix.ini','global/ORFix.ini','global/ORFix/ORFix.ini')) and not any(p.name.lower()=='orfix.ini' for p in root.rglob('*.ini')):out.append('未找到 ORFix 全局库（可能位于扫描范围外）')
    if b'commandlist\\texfx\\' in content and not any((p/q).is_file() for p in ancestors for q in ('TexFx/Main.ini','global/TexFx/Main.ini','TexFx.ini')):out.append('未找到 TexFx 全局库（可能位于扫描范围外）')
    return out

def run(folder,phase,prior=None):
    root=Path(folder).resolve();r=Result(phase);when=stamp()
    if not root.is_dir():r.errors.append('无效目录');return r
    if phase not in ('preview','normal','face','rollback'):raise ValueError('未知操作')
    if phase=='rollback':
        try:r.modified,r.changes=restore(root)
        except Exception as e:r.errors.append(str(e))
    else:
        plans,r.scanned,r.errors=scan(root,phase=='face')
        r.records=[x for p in plans for x in p.records];r.warnings=env_warnings(root,plans)
        if phase in ('normal','face'):
            # Atomic per whole invocation; avoid partial commits across INI/BUF.
            edits=[]
            for p in plans:
                if p.patched!=p.original:edits.append((p.path,p.original,p.patched));edits.extend(p.companions)
            if edits:
                entries=[]
                try:
                    # Check stage continuity BEFORE changing files.
                    if phase=='face' and prior:
                        for p,a,b in edits:
                            old=next((x for x in prior if x['ini']==rel(root,p)),None)
                            if old and sha(a)!=old['patched_sha256']:raise ValueError('面部阶段前文件已被修改：'+str(p))
                    entries=commit(root,edits,when)
                    merged=merge(prior or [],entries) if phase=='face' else entries
                    manifest(root,merged,when);r.entries=merged;r.modified=len(entries)
                    backs={str(root/e['ini']):e['backup'] for e in entries}
                    for d in r.records:
                        if d.decision in ('auto_repair','requires_confirmation') and d.file in backs:d.execution='written';d.backup=backs[d.file]
                    r.changes=['已修复：'+e['ini'] for e in entries]
                except Exception as e:
                    if entries:
                        for p,a,b in reversed(edits):
                            if p.read_bytes()==b:atomic(p,a)
                    r.errors.append(str(e))
            elif phase=='face':r.entries=prior or []
        else:r.changes=['候选：'+rel(root,p.path) for p in plans if p.original!=p.patched]
        if phase in ('normal','preview'):
            # Preview uses prospective ordinary output, normal uses current disk.
            for p in plans:
                f=face_plan(p.path,p.patched if phase=='preview' else p.path.read_bytes());r.records.extend(f.records)
                r.face_risks += [f'{rel(root,p.path)} | {x.section}: {x.reason or x.rule_id}' for x in f.records if x.decision in ('requires_confirmation','needs_diagnosis')]
    try:r.log_path=write_reports(root,when,phase,r.records,{'scanned':r.scanned,'modified':r.modified,'face_risks':r.face_risks,'changes':r.changes,'errors':r.errors,'warnings':r.warnings})
    except OSError as e:r.errors.append('日志写入失败：'+str(e))
    return r
