"""Lossless line-indexed 3DMigoto INI model. Unknown commands remain opaque."""
from dataclasses import dataclass, field
from pathlib import Path, PureWindowsPath
from collections import defaultdict
import re

HEADER=re.compile(r'^\s*\[([^\]\r\n]+)\]\s*(?:;.*)?$')
BIND=re.compile(r'^([ \t]*)(;[ \t]*)?ps-t(\d+)[ \t]*=[ \t]*(Resource[A-Za-z0-9_.-]+)[ \t]*(?:;.*)?$',re.I)
RUN=re.compile(r'^run\s*=\s*(.+)$',re.I)
BRANCH=re.compile(r'^(?:if\b|else\b|elif\b|endif\b)',re.I)
DRAW=re.compile(r'^(?:drawindexed|drawindexedinstanced|draw)\s*=',re.I)
FIX=re.compile(r'^run\s*=\s*CommandList\\global\\ORFix\\(NNFix|ORFix)$',re.I)
HASH=re.compile(r'[0-9a-f]{8}',re.I)
REF=re.compile(r'Resource[A-Za-z0-9_.-]+',re.I)
ORFIX=r'run = CommandList\global\ORFix\ORFix'
NNFIX=r'run = CommandList\global\ORFix\NNFix'

def code(s): return s.split(';',1)[0].strip().lstrip('\ufeff')
def kv(s):
    a,sep,b=code(s).partition('='); return (a.strip().lower(),b.strip()) if sep else ('','')
def tokens(s): return [x.lower() for x in re.findall(r'[A-Z]?[a-z]+|[A-Z]+|[0-9]+',s)]
def variant(ref, strict=False):
    r=ref.lower(); match=re.search(r'\.\d+$' if strict else r'[._-]?\d+$',r)
    suffix=match[0] if match else ''; r=r[:match.start()] if match else r
    for k in ('normalmap','diffuse','lightmap','metalmap','shadowramp','outlines','outline'):
        if r.endswith(k) and len(r)>len(k): return r[:-len(k)],'outline' if k=='outlines' else k,suffix
    return '', '', ''
@dataclass
class Section:
    name:str; start:int; end:int
    @property
    def key(self):return self.name.lower()
@dataclass
class Patch:
    before:dict=field(default_factory=dict); after:dict=field(default_factory=dict); replace:dict=field(default_factory=dict); remove:set=field(default_factory=set)
    def add(self,where,line,text): getattr(self,where).setdefault(line,[]).append(text)
    def apply(self,d):
        out=[]
        for i,l in enumerate(d.lines):
            out.extend(x+d.cr for x in self.before.get(i,[]))
            if i not in self.remove:out.append(self.replace.get(i,l))
            out.extend(x+d.cr for x in self.after.get(i,[]))
        return '\n'.join(out).encode('utf-8')
@dataclass
class Doc:
    path:Path; data:bytes
    def __post_init__(self):
        self.lines=self.data.decode('utf-8').split('\n');self.cr='\r' if b'\r\n' in self.data else ''
        self.sections=[]; self.by={};self.duplicates=set(); self.resources={}; self.graph={}
        for i,l in enumerate(self.lines):
            m=HEADER.fullmatch(l.rstrip('\r').lstrip('\ufeff') if i==0 else l.rstrip('\r'))
            if m:
                if self.sections:self.sections[-1].end=i
                s=Section(m[1].strip(),i,len(self.lines));self.sections.append(s)
                if s.key in self.by:self.duplicates.add(s.key)
                self.by[s.key]=s
        for s in self.sections:
            if s.key.startswith('resource'):
                names=[v.strip('"') for _,k,v in self.items(s) if k=='filename']
                if len(names)==1 and s.key not in self.duplicates:self.resources[s.key]=names[0]
            self.graph[s.key]=[v.lower() for _,k,v in self.items(s) if k=='run' and v.lower().startswith('commandlist')]
    def active(self,s):return [(i,code(self.lines[i])) for i in range(s.start+1,s.end) if code(self.lines[i])]
    def items(self,s):return [(i,*kv(l)) for i,l in self.active(s)]
    def value(self,s,k):
        vs=[v for _,key,v in self.items(s) if key==k]; return vs[0] if len(vs)==1 else ''
    def namespace(self):return next((v.lower() for l in self.lines for k,v in [kv(l)] if k=='namespace'),'')
    def is_library(self):
        n=self.namespace();return n in ('texfx','gimi','orfix') or n.startswith('global\\') or 'shaderfix' in n
    def unsafe_global(self):return bool(self.is_library() or self.duplicates or any(k in ('include','include_recursive') for l in self.lines for k,v in [kv(l)]) or any(s.key.startswith('shaderregex') for s in self.sections))
    def legacy(self):return any(s.key in ('shaderregexcharreflection','commandlistreflectiontexture') for s in self.sections)
    def restricted(self):return bool(self.namespace() or self.duplicates or any(k in ('include','include_recursive') for l in self.lines for k,v in [kv(l)]) or any(s.key.startswith('shaderregex') for s in self.sections))
    def file(self,ref):
        name=self.resources.get(ref.lower(),''); return safe_file(self.path.parent,name)
    def reachable(self,roots):
        todo=list(roots); seen=set();missing=set()
        while todo:
            n=todo.pop()
            if n in seen:continue
            seen.add(n)
            for x in self.graph.get(n,[]):
                if x in self.by and x.startswith('commandlist'):todo.append(x)
                elif not x.startswith(('commandlist\\global\\orfix\\','commandlist\\texfx\\','commandlist\\global\\texfx\\','commandlist\\global\\gimi\\')):missing.add(x)
        return seen,missing

def safe_file(root,name):
    if not name:return None
    w=PureWindowsPath(name); p=Path(name.replace('\\','/'))
    if w.drive or w.root or p.is_absolute() or '..' in p.parts:return None
    p=root/p
    try:
        if p.is_symlink() or not p.resolve().is_relative_to(root.resolve()) or not p.is_file() or p.stat().st_size<=0:return None
        return p
    except OSError:return None

def strict_file(root,name,ext):
    p=safe_file(root,name)
    if not p or any(c in name for c in '\\/:') or p.suffix.lower()!=ext:return None
    if ext=='.dds':
        try:
            with p.open('rb') as f: h=f.read(8)
            if h!=b'DDS |\x00\x00\x00' or p.stat().st_size<128:return None
        except OSError:return None
    return p

def valid(s):
    if not s.key.startswith('textureoverride'):return False
    tail=s.name[len('TextureOverride'):]; ts=tokens(tail)
    bad={'face','gun','weapon','icon','shadow','outline','effect','eyebrow','pupil','glow','fix'}
    return not (set(ts)&bad or len(ts)<=1 and any(x in tail.lower() for x in bad-{'gun'}))
def main_stem(d,stem):return 'textureoverride'+stem+'head' in d.by and any('textureoverride'+stem+x in d.by for x in ('body','dress','extra'))
def outfit(d,s):
    if not valid(s):return False
    tail=s.key[len('textureoverride'):]
    if tail.endswith(('body','dress','extra')):return True
    if not tail.endswith('head'):return False
    stem=tail[:-4]
    if main_stem(d,stem):return True
    for x in ('eyes','eye','hair'):
        if stem.endswith(x) and main_stem(d,stem[:-len(x)]):return bool(d.file(d.value(s,'ib')))
    return False

def classify(d,bindings):
    # bindings: (line, slot, ref), same material group only
    slots={n:r for _,n,r in bindings}
    if len(slots)!=len(bindings):return '', 'duplicate texture slot'
    layout={ (0,1):('diffuse','lightmap'),(0,1,2):('normalmap','diffuse','lightmap'),(0,1,2,3):('diffuse','lightmap','metalmap','shadowramp')}.get(tuple(sorted(slots)))
    if not layout:return '', 'unsupported map layout / additional texture slot'
    types=[variant(slots[i],True) for i in range(len(layout))]
    if tuple(t[1] for t in types)!=layout or len({(t[0],t[2]) for t in types})!=1:return '', 'texture stem/variant or type mismatch'
    if any(not d.file(r) for r in slots.values()):return '', 'missing resources'
    return ('ORFix' if len(layout)==3 else 'NNFix'),''
