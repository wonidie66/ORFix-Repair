"""Bounded, auditable subset of 3DMigoto control flow; never evaluates arbitrary code."""
from .parser import *
from dataclasses import dataclass
DECL=re.compile(r'global\s+persist\s+(\$[A-Za-z_]\w*)\s*=\s*(-?\d+)',re.I)
EQ=re.compile(r'(\$[A-Za-z_]\w*)\s*==\s*(-?\d+)')
GUARD=re.compile(r'\$[A-Za-z_]\w*\s*(?:==|!=|<=|>=|<|>)\s*-?\d+(?:\s*(?:&&|\|\|)\s*\$[A-Za-z_]\w*\s*(?:==|!=|<=|>=|<|>)\s*-?\d+)*')
DRAW_INDEX=re.compile(r'\d+\s*,\s*\d+\s*,\s*-?\d+')

def domains(d):
    init={};cycles={};bad=set()
    for s in d.sections:
        for i,l in d.active(s):
            m=DECL.fullmatch(l)
            if m and s.key=='constants':
                n=m[1].lower()
                if n in init:bad.add(n)
                init[n]=int(m[2]);continue
            k,v=kv(l);k=k.removeprefix('post ').strip()
            if not re.fullmatch(r'\$[A-Za-z_]\w*',k):continue
            if not s.key.startswith('key') or d.value(s,'type').lower()!='cycle' or k in cycles:bad.add(k);continue
            try:
                vals=[int(x.strip()) for x in v.split(',')]
                if len(vals)!=len(set(vals)) or len(vals)>64:bad.add(k)
                cycles[k]=vals
            except ValueError:bad.add(k)
    return {k:v for k,v in cycles.items() if k not in bad and k in init and init[k] in v}
@dataclass
class Group:
    slot:int; bindings:list; variable:str=''; values:list=None; has_else:bool=False

def read_group(d,active,pos,res,dom,face=False):
    def assignment(i,l):
        k,v=kv(l);slot=0 if face and k=='this' else int(k[4:]) if re.fullmatch(r'ps-t\d+',k) else -1
        return (i,slot,v,res[v.lower()]) if slot in (0,1,2,70) and v.lower() in res else None
    if pos>=len(active):return None,pos
    a=assignment(*active[pos])
    if a:return Group(a[1],[a]),pos+1
    binds=[];values=[];var='';has_else=False;closed=False
    while pos<len(active):
        i,l=active[pos]; low=l.lower()
        if not binds and low.startswith('if '):guard=l[3:]
        elif binds and not has_else and low.startswith('else if '):guard=l[8:]
        elif binds and not has_else and low=='else':guard='';has_else=True
        else:return None,pos
        if guard:
            m=EQ.fullmatch(guard.strip())
            if not m or var and var.lower()!=m[1].lower() or int(m[2]) in values:return None,pos
            var=m[1];values.append(int(m[2]))
        pos+=1
        if pos>=len(active):return None,pos
        a=assignment(*active[pos])
        if not a or binds and a[1]!=binds[0][1]:return None,pos
        binds.append(a);pos+=1
        if pos>=len(active):return None,pos
        if active[pos][1].lower()=='endif':pos+=1;closed=True;break
        if has_else:return None,pos
    if not closed or len(binds)<2 or not has_else and (not dom.get(var.lower()) or not set(dom[var.lower()])<=set(values)):return None,pos
    return Group(binds[0][1],binds,var,values,has_else),pos

def disjoint(a,b):
    if '||' in a+b:return False
    x={m[1]:int(m[2]) for v in a.split('&&') if (m:=EQ.fullmatch(v.strip()))}
    return any(m[1] in x and x[m[1]]!=int(m[2]) for v in b.split('&&') if (m:=EQ.fullmatch(v.strip())))

def draw_only(active,allow_repair=False):
    stack=[];draws=0;prev_guard='';prev_closed=False;replace={};conditional=False
    for i,l in active:
        low=l.lower()
        if low.startswith('if ') or low.startswith('else if ') or low.startswith('elif '):
            initial=low.startswith('if ');g=l[3:] if initial else l[8:] if low.startswith('else if ') else l[5:]
            if allow_repair:
                if not GUARD.fullmatch(g.strip()):return None
            elif not re.fullmatch(r'[A-Za-z0-9_$\s()!<>=&|+\-]+',g) or '$' not in g or not any(c in g for c in '=<>!') or g.count('(')!=g.count(')'):return None
            conditional=True
            if initial:stack.append([g,False,False])
            elif not stack:
                if not allow_repair or not low.startswith('else if ') or not prev_closed or not disjoint(prev_guard,g):return None
                replace[i]=True;stack.append([g,False,False])
            elif stack[-1][1]:return None
            else:stack[-1][2]=True
            prev_closed=False
        elif low=='else':
            if not stack or stack[-1][1]:return None
            stack[-1][1:]=[True,True];prev_closed=False
        elif low=='endif':
            if not stack:return None
            g,_,had=stack.pop();prev_closed=not stack and not had;prev_guard=g
        else:
            k,v=kv(l)
            if k!='drawindexed' or not (DRAW_INDEX.fullmatch(v) or not allow_repair and v=='0'):return None
            draws+=v!='0';prev_closed=False
    return (conditional,replace) if draws and not stack else None
