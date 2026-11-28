"""V4.2.1 complex static and dynamic contracts; strict local DDS and IB checks."""
from ..parser import *
from ..controlflow import *
from ..diagnostics import decision
from dataclasses import dataclass
@dataclass
class Resource:
    kind:str;stem:str;stray:int;name:str

def resources(d):
    out={}
    for s in d.sections:
        if not REF.fullmatch(s.name):continue
        filename='';stray=-1;good=True
        for i,l in d.active(s):
            k,v=kv(l)
            if k=='filename' and not filename:filename=v
            elif re.fullmatch(r'[A-Za-z]',l) and stray<0:stray=i
            else:good=False;break
        if not good or not strict_file(d.path.parent,filename,'.dds'):continue
        st,kind,_=variant(Path(filename).stem);rs,rk,_=variant(s.name)
        if kind and kind==rk and rs:out[s.key]=Resource(kind,st,stray,filename)
    return out

def verify_ib(d,ref):
    s=d.by.get(ref.lower())
    if not s or not REF.fullmatch(ref):return False
    items=d.items(s);names=[v for _,k,v in items if k=='filename']
    return len(names)==1 and all(k=='filename' or k=='type' and v.lower()=='buffer' or k=='format' and v.lower() in ('dxgi_format_r32_uint','dxgi_format_r16_uint') for _,k,v in items) and bool(strict_file(d.path.parent,names[0],'.ib'))

def linked_mesh(d,s,meta):
    part=s.key.removeprefix('textureoverride')
    if any(k in part for k in ('face','eye','pupil','weapon','shadow','outline','effect','icon','glow')) or meta.get('ib','').lower()!='resource'+part+'ib' or not verify_ib(d,meta.get('ib','')):return ''
    for p in d.sections:
        if not p.key.startswith('textureoverride') or not p.key.endswith('ib') or d.value(p,'hash').lower()!=meta['hash'].lower() or d.value(p,'handling').lower()!='skip':continue
        stem=p.key[len('textureoverride'):-2]
        if not stem or not part.startswith(stem) or not re.fullmatch(r'[a-z0-9_.-]*',part[len(stem):]):continue
        good=True
        for k in ('position','blend','texcoord'):
            x=d.by.get('textureoverride'+stem+k);r=d.by.get('resource'+stem+k)
            if not x or not r or not HASH.fullmatch(d.value(x,'hash')) or d.value(x,'vb0' if k=='position' else 'vb1').lower()!=r.key or not strict_file(d.path.parent,d.value(r,'filename'),'.buf'):good=False;break
        if good:return stem
    return ''

def transform(d,dynamic=False):
    p=Patch();logs=[]
    if d.restricted():return d.data,logs
    res=resources(d);dom=domains(d)
    for s in d.sections:
        if not s.key.startswith('textureoverride'):continue
        active=d.active(s);meta={};pos=0;good=True
        # Static legacy permits metadata interspersed before draws; actual
        # complex sample has it first. Dynamic rule requires initial metadata.
        while pos<len(active):
            k,v=kv(active[pos][1])
            if k not in ('hash','match_first_index','ib','handling'):break
            if k in meta:good=False;break
            meta[k]=v;pos+=1
        if not good or not HASH.fullmatch(meta.get('hash','')) or not re.fullmatch(r'\d+',meta.get('match_first_index','')) or int(meta['match_first_index'])>0xffffffff or meta.get('handling','skip').lower()!='skip':continue
        if dynamic:
            stem=linked_mesh(d,s,meta)
            if not stem:continue
        else:
            tail=s.key.removeprefix('textureoverride');kind=next((k for k in ('head','body','dress','extra') if tail.endswith(k)),'')
            if not kind or not main_stem(d,tail[:-len(kind)]) or not valid(s) or any(x in tail for x in ('eye','pupil')) or not verify_ib(d,meta.get('ib','')):continue
            stem=''
        groups={};has_dynamic=False
        while pos<len(active):
            g,n=read_group(d,active,pos,res,dom)
            if not g or not dynamic and g.variable:break
            if g.slot in groups:good=False;break
            groups[g.slot]=g;has_dynamic|=bool(g.variable);pos=n
        if not good or set(groups) not in ({0,1,2},{0,1,2,70}):continue
        texfx=False
        if dynamic and pos<len(active) and active[pos][1].lower()==r'run = commandlist\texfx\t.1':
            if 70 not in groups:continue
            texfx=True;pos+=1
        if pos>=len(active):continue
        draw=draw_only(active[pos:],allow_repair=not dynamic)
        if not draw:continue
        conditional,syntax=draw;types={};core=set();refstems=set();guards=[];q=Patch()
        for slot,g in groups.items():
            actual=g.bindings[0][3].kind
            if actual in types or (slot==70)!=(actual=='outline'):good=False;break
            types[actual]=g
            for i,_,ref,r in g.bindings:
                rs,_,_=variant(ref)
                if r.kind!=actual or dynamic and (r.stray>=0 or rs.removeprefix('resource')!=r.stem):good=False
                if slot!=70 or not dynamic:core.add(r.stem);refstems.add(rs)
                if dynamic and slot!=70 and not r.stem.startswith(stem):good=False
            if g.variable and not g.has_else:
                expr='('+' || '.join(f'{g.variable} == {v}' for v in g.values)+')'
                if expr not in guards:guards.append(expr)
        if not good or len(core)!=1 or not dynamic and len(refstems)!=1 or not {'normalmap','diffuse','lightmap'}<=types.keys():continue
        reorder=any(groups[n].bindings[0][3].kind!=k for n,k in enumerate(('normalmap','diffuse','lightmap')))
        if dynamic and reorder and has_dynamic:continue
        if not dynamic and not conditional and not reorder and len(groups)==3:continue
        if reorder:
            for n,k in enumerate(('normalmap','diffuse','lightmap')):
                if dynamic:
                    for i,slot,ref,r in types[k].bindings:
                        if slot!=n:q.replace[i]=re.sub(r'(?i)ps-t\d+',f'ps-t{n}',d.lines[i],count=1)
                else:
                    i,_,ref,r=groups[n].bindings[0];target=types[k].bindings[0][2]
                    if ref!=target:q.replace[i]=re.sub(re.escape(ref),lambda _:target,d.lines[i],count=1)
        if not dynamic:
            for i in syntax:q.replace[i]=re.sub(r'(?i)else if','if',d.lines[i],count=1)
            for g in groups.values():
                for i,slot,ref,r in g.bindings:
                    if r.stray>=0:q.remove.add(r.stray)
        anchor=active[pos][0]
        if guards:
            q.add('before',anchor,'if '+' && '.join(guards));q.add('before',anchor,'    '+ORFIX);q.add('before',anchor,'endif')
        else:q.add('before',anchor,ORFIX)
        logs.append(decision(d,s,'MAT-006' if dynamic else 'MAT-004','auto_repair',['IB 与真实资源核验','三贴图语义完整','分支与绘制路径核验',f'辅助 Outline 槽位：{70 in groups}',f'保留 TexFx T.1：{texfx}',f'动态分支保护：{guards}',f'槽位重排：{reorder}',f'局部语法修正：{len(syntax)}'],patch=q.apply(d)))
        p.before.update(q.before);p.replace.update(q.replace);p.remove.update(q.remove)
    return p.apply(d),logs
