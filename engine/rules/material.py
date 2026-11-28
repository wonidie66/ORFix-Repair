"""Conservative ordinary and enhanced material rules, ported from verified contracts."""
from ..parser import *
from ..diagnostics import decision
from collections import defaultdict

TEX_RUN=re.compile(r'run\s*=\s*CommandList\\TexFx\\Transparency(?:\.[01])?',re.I)

def bound(d,s):
    out=[]
    for i,l in d.active(s):
        m=BIND.fullmatch(l)
        if m and not m[2]:out.append((i,int(m[3]),m[4]))
    return out

def simple_shader(d,s):
    calls=[v.lower() for _,k,v in d.items(s) if k=='run']
    sh=d.by.get('customshadertransparency')
    if calls!=['customshadertransparency'] or not sh:return False
    data={k:v.lower() for _,k,v in d.items(sh)}
    return data.get('blend')=='add blend_factor inv_blend_factor' and data.get('handling')=='skip' and data.get('drawindexed')=='auto' and all(k in ('blend','handling','drawindexed') or k.startswith('blend_factor[') for k in data)

def record(d,s,r,p,checks):return decision(d,s,r,'auto_repair',checks,patch=p.apply(d))

def basic(d):
    p=Patch();logs=[]
    if d.restricted():return d.data,logs
    for s in d.sections:
        if not valid(s) or not any(k in ('ib','match_first_index') for _,k,v in d.items(s)):continue
        if not outfit(d,s):continue
        active=d.active(s);calls=[v for _,k,v in d.items(s) if k=='run']
        if any(BRANCH.match(l) for _,l in active):continue
        entries=defaultdict(list);other=False
        for i in range(s.start+1,s.end):
            l=d.lines[i].rstrip('\r');m=BIND.fullmatch(l)
            if m:
                if int(m[3])>2:other=True
                else:entries[int(m[3])].append((i,m[4],bool(m[2])))
            elif code(l).lower().startswith('ps-t'):other=True
        existing=[m[1].lower() for _,l in active if (m:=FIX.fullmatch(l))]
        if other or 'nnfix' in existing:continue
        q=Patch();rid='MAT-002'
        if not entries[2]:
            if existing or len(entries[0])!=1 or len(entries[1])!=1 or any(e[2] for n in (0,1) for e in entries[n]):continue
            a,b=entries[0][0],entries[1][0]
            # Legacy base does not accept suffix variants; enhanced classifier does.
            if not a[1].lower().endswith('diffuse') or a[1][:-7].lower()+'lightmap'!=b[1].lower() or not d.file(a[1]) or not d.file(b[1]) or calls and not simple_shader(d,s):continue
            q.add('after',b[0],NNFIX);rid='MAT-001'
        else:
            if len(entries[2])!=1 or entries[2][0][2]:continue
            i,ref,_=entries[2][0]
            if not ref.lower().endswith('lightmap'):continue
            expect=[ref[:-8]+'NormalMap',ref[:-8]+'Diffuse']
            if not all(d.file(x) for x in [*expect,ref]):continue
            # Protect unknown run and explicit incompatible custom shader. Existing
            # historical ORFix can still have commented slot restoration.
            first_binding=min(e[0] for es in entries.values() for e in es)
            if any(not FIX.fullmatch(l) and not (j<first_binding and re.fullmatch(r'run\s*=\s*CommandList[A-Za-z0-9_.-]+',l,re.I)) for j,l in active if RUN.match(l)):continue
            good=True
            for slot,x in enumerate(expect):
                live=[e for e in entries[slot] if not e[2]];comment=[e for e in entries[slot] if e[2]]
                if len(live)>1 or len(comment)>1 or any(e[1].lower()!=x.lower() for e in live+comment):good=False;break
                if not live:
                    if comment:
                        j=comment[0][0];raw=d.lines[j];ind=raw[:len(raw)-len(raw.lstrip(' \t'))];q.replace[j]=ind+raw[len(ind):].lstrip('; \t')
                    elif entries[0] or entries[1]:q.add('before',i,re.match(r'[ \t]*',d.lines[i])[0]+f'ps-t{slot} = {x}')
                    else:good=False;break
            if not good:continue
            if 'orfix' not in existing:q.add('after',i,ORFIX)
        if q.apply(d)==d.data:continue
        logs.append(record(d,s,rid,q,['角色网格关联','贴图名称及布局匹配','实际文件非空','保留原绘制与资源']))
        p.before.update(q.before);p.after.update(q.after);p.replace.update(q.replace)
    return p.apply(d),logs

def safe_texfx(d):
    p=Patch();logs=[]
    if d.restricted():return d.data,logs
    for s in d.sections:
        if not outfit(d,s) or not any(k in ('ib','match_first_index') for _,k,v in d.items(s)):continue
        active=d.active(s); calls=[(i,l) for i,l in active if RUN.match(l)]
        if not any(TEX_RUN.fullmatch(l) for _,l in calls):continue
        if any(FIX.fullmatch(l) for _,l in calls):continue
        b=bound(d,s); mask=[x for x in b if x[1]==69];core=[x for x in b if x[1]!=69]
        kind,reason=classify(d,core)
        if any(BRANCH.match(l) for _,l in active) or len(calls)!=1 or len(mask)!=1 or not d.file(mask[0][2]) or kind!='NNFix' or len(core)!=2 or any(k.startswith('ps-t') and not BIND.fullmatch(code(d.lines[i])) for i,k,v in d.items(s)):
            logs.append(decision(d,s,'TEX-001','needs_diagnosis',[],reason or '非单一透明调用/遮罩或存在复杂条件'));continue
        anchor=max(mask[0][0],core[-1][0]);
        if anchor>=calls[0][0]:continue
        q=Patch();q.add('after',anchor,NNFIX);p.after.update(q.after)
        logs.append(record(d,s,'TEX-001',q,['双贴图布局','遮罩资源存在','单一 TexFx Transparency','无条件分支']))
    return p.apply(d),logs

def enhanced(d):
    p=Patch();logs=[]
    if d.unsafe_global():return d.data,logs
    roots={s.key for s in d.sections if outfit(d,s)}; reach,_=d.reachable(roots)
    # Shared local commands can affect unrecognized / non-character draws.
    other,_=d.reachable({s.key for s in d.sections if s.key.startswith(('textureoverride','shaderoverride','customshader','present')) and s.key not in roots})
    hastex=any('commandlist\\texfx\\' in code(l).lower() for l in d.lines)
    for s in d.sections:
        if s.key not in reach or s.key.startswith('commandlist') and s.key in other:continue
        if set(tokens(s.name)) & {'face','weapon','gun','sword','icon','outline','shadow','particle','weaponfx'}:continue
        # Legacy enhanced flushes binding groups on branches, calls, draws. We do
        # not carry an unproven resource across a branch or foreign command.
        group=[];hasfix=False;mask=-1;inserted=False
        def flush(boundary):
            nonlocal group,hasfix,mask,inserted
            if not group:return
            kind,reason=classify(d,group)
            if kind and not hasfix and not inserted:
                anchor=max(group[-1][0],mask);ind=re.match(r'[ \t]*',d.lines[group[0][0]])[0]
                q=Patch()
                if s.key.startswith('commandlist') and hastex and mask<0:q.add('after',anchor,ind+'ps-t69 = null')
                q.add('after',anchor,ind+(ORFIX if kind=='ORFix' else NNFIX));p.after.update(q.after)
                rid='MAT-003' if len(group)==4 else 'MAT-007' if s.key.startswith('commandlist') or any(BRANCH.match(l) for _,l in d.active(s)) else 'MAT-002' if kind=='ORFix' else 'MAT-001'
                logs.append(record(d,s,rid,q,['独立绑定组','资源语义及同 stem/variant','资源文件存在','绘制/命令边界前']))
                inserted=True
            elif not kind and {x[1] for x in group}&{0,1}:logs.append(decision(d,s,'MAT-007','needs_diagnosis',[],reason))
            elif kind and hasfix:logs.append(decision(d,s,'MAT-007','no_change',['当前绑定组已有修复调用']))
            group=[];hasfix=False;mask=-1
        for i,l in d.active(s):
            if BRANCH.match(l):flush(i);inserted=False;continue
            if FIX.fullmatch(l):hasfix=True;continue
            k,v=kv(l)
            if k=='ps-t69':mask=i;continue
            m=BIND.fullmatch(l)
            if m:
                slot=int(m[3]);
                if slot==0 and group:flush(i);inserted=False
                group.append((i,slot,m[4]));hasfix=False;continue
            if k.startswith('ps-t'):
                # Opaque resource assignment invalidates the group.
                group=[];hasfix=False;mask=-1;continue
            if RUN.match(l) or DRAW.match(l):flush(i)
        flush(s.end)
    return p.apply(d),logs
