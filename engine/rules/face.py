"""Face fixes are plans only; the application requires an explicit second phase."""
from ..parser import *
from ..controlflow import domains,read_group
from .complex import resources
from ..diagnostics import decision

def is_face(s):return s.key.startswith('textureoverride') and 'face' in s.key and 'diffuse' in s.key

def combo(d):
    p=Patch();logs=[]
    if d.restricted():return d.data,logs
    res=resources(d);dom=domains(d)
    for s in d.sections:
        if not is_face(s) or not s.key.endswith('diffuse'):continue
        active=d.active(s)
        if not active or kv(active[0][1])[0]!='hash' or not HASH.fullmatch(kv(active[0][1])[1]):continue
        g,pos=read_group(d,active,1,res,dom,True)
        if not g or g.slot!=0 or len(g.bindings)<2 or pos+1!=len(active):continue
        l,n=read_group(d,active,pos,res,dom)
        if not l or l.slot!=1 or len(l.bindings)!=1 or l.variable:continue
        i,slot,ref,r=l.bindings[0]
        stem=s.key[len('textureoverride'):-len('diffuse')];prefix=stem.split('face',1)[0]
        if not prefix or r.kind!='lightmap' or r.stray>=0 or not variant(ref)[0].startswith('resource'+prefix+'face') or not r.stem.startswith(prefix+'face'):continue
        keys={kv(d.lines[x[0]])[0] for x in g.bindings}
        if len(keys)!=1 or any(a[3].kind!='diffuse' or a[3].stray>=0 or variant(a[2])[0]!='resource'+stem or a[3].stem!=stem for a in g.bindings):continue
        q=Patch()
        for j,_,_,_ in g.bindings:
            if keys=={'ps-t0'}:q.replace[j]=re.sub(r'(?i)ps-t0','this',d.lines[j],count=1)
        raw=d.lines[i];ind=re.match(r'[ \t]*',raw)[0];q.replace[i]=ind+'; '+raw[len(ind):]
        logs.append(decision(d,s,'FACE-002+003','requires_confirmation',['Hash 格式正确','多变体面部 Diffuse','互斥分支完整','本地 DDS 头及资源组核验','单一无条件面部 LightMap','无额外指令'], 'B 版组合实测有效；不证明 LightMap 单独为根因',q.apply(d)))
        p.replace.update(q.replace)
    return p.apply(d),logs

def legacy(d):
    p=Patch();logs=[]
    if d.is_library() or d.duplicates:return d.data,logs
    roots={s.key for s in d.sections if is_face(s)}
    allface,_=d.reachable(roots)
    other,_=d.reachable({s.key for s in d.sections if s.key.startswith(('textureoverride','shaderoverride','customshader','present')) and s.key not in roots})
    callers={}
    for n,targets in d.graph.items():
        for x in targets:callers.setdefault(x,set()).add(n)
    proposals={};blocked=set()
    for key in sorted(roots):
        s=d.by[key];reach,missing=d.reachable([key]);reasons=[];ed={}
        direct=any(k=='ps-t0' for _,k,v in d.items(s));
        for name in sorted(reach):
            sec=d.by[name];items=d.items(sec);t0=[(i,v) for i,k,v in items if k=='ps-t0'];hasthis=any(k=='this' for _,k,v in items)
            if not t0:
                if direct and hasthis and name!=key:reasons.append('调用链另有 this')
                continue
            if name!=key and (name in other or callers.get(name,set())-allface):reasons.append('共享 CommandList: '+sec.name);continue
            if hasthis:reasons.append('ps-t0 与 this 混用: '+sec.name);continue
            for i,ref in t0:
                if not REF.fullmatch(ref) or 'face' not in ref.lower() or variant(ref)[1]!='diffuse' or not d.file(ref) or Path(d.resources[ref.lower()]).suffix.lower() not in ('.dds','.tga','.png','.jpg','.jpeg','.bmp'):
                    reasons.append('不能确认面部贴图：'+ref);continue
                ed[i]=re.sub(r'(?i)ps-t0','this',d.lines[i],count=1)
        if missing:reasons.append('无法解析调用链：'+', '.join(sorted(missing)))
        if reasons:
            blocked.update(ed);logs.append(decision(d,s,'FACE-001','needs_diagnosis',[], '；'.join(reasons)));continue
        if ed:proposals[key]=ed
    for key,ed in proposals.items():
        s=d.by[key]
        if set(ed)&blocked:logs.append(decision(d,s,'FACE-001','needs_diagnosis',[], '其他面部根的调用链存在歧义'));continue
        q=Patch(replace=ed);p.replace.update(ed)
        logs.append(decision(d,s,'FACE-001','requires_confirmation',['面部 Diffuse 专用资源','文件存在','非共享调用链','无 this 冲突'],patch=q.apply(d)))
    return p.apply(d),logs

def analyze(d):
    out,logs=combo(d);out2,more=legacy(Doc(d.path,out));return out2,logs+more
