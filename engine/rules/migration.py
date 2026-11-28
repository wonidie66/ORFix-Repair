"""Exact historical recipes, not a universal hash update database."""
from ..parser import *
from ..diagnostics import decision
import hashlib,struct,math
REMAP=[70, 6, 20, 21, 22, 12, 13, 14, 15, 16, 7, 17, 18, 19, 8, 9, 10, 11, 4, 5, 69, 54, 53, 94, 111, 115, 71, 88, 92, 91, 52, 114, 116, 117, 118, 95, 93, 67, 72, 65, 68, 60, 61, 62, 63, 64, 66, 55, 56, 57, 58, 59, 40, 44, 45, 32, 23, 29, 24, 33, 37, 38, 30, 89, 90, 73, 79, 76, 82, 85, 86, 74, 75, 77, 78, 80, 81, 83, 84, 87, 2, 0, 112, 113, 96, 102, 99, 105, 108, 109, 97, 98, 100, 101, 103, 104, 106, 107, 110, 41, 43, 48, 28, 3, 1, 31, 36, 35, 34, 26, 27, 25, 39, 49, 51, 47, 46, 42, 50]
PAIRS={'55b43e99':('f9e1b52b',1),'9f7dc19c':('735eaea4',1),'906ad233':('1f0ab400',1),'abdc3768':('ad75352c',4),'6d197625':('4c2f9a0a',1)}
HASHLINE=re.compile(r'^([ \t]*hash[ \t]*=[ \t]*)([0-9a-f]{8})([ \t]*(?:;[^\r\n]*)?\r?)$',re.I|re.M)
CIT_ORIG={'2df4453b1ca0f1537d6c9a030bd94f99adf5425b4cc274444b0eb4227c019d06','8b4f568f4aac0f4db3fbeb253408a7925195b65426b5721f3db833eed041140f'}
CIT_FIXED='50436707bfeab618a9b4d64717d9540d030477f6e23763706654147582a87b99'
def recipe(d):
    if d.path.name.lower()!='citlali.ini':return d.data,[],False
    h=hashlib.sha256(d.data).hexdigest()
    if h==CIT_FIXED:return d.data,[],True
    if h not in CIT_ORIG:return d.data,[],False
    s=Section('Citlali exact recipe',0,len(d.lines))
    if not all(safe_file(d.path.parent,n) for n in ['CitlaliBody.ib','CitlaliBodyTexfx.dds','CitlaliBodyDiffuse.dds','CitlaliBodyLightMap.dds']):return d.data,[decision(d,s,'TEX-RECIPE-001','needs_diagnosis',[],'专项资源缺失')],True
    out=(Path(__file__).resolve().parents[1]/'known_fixes/Citlali_high_priest_verified_v2.ini').read_bytes()
    if hashlib.sha256(out).hexdigest()!=CIT_FIXED:raise ValueError('专项资源 SHA-256 不匹配')
    return out,[decision(d,s,'TEX-RECIPE-001','auto_repair',['精确 SHA-256','必需资源存在'],patch=out)],True

def remap_blend(src):
    if not src or len(src)%32:raise ValueError('Blend.buf 非 32 字节记录')
    dst=bytearray(src);count=0
    for i in range(0,len(src),32):
        for k in range(4):
            w=struct.unpack_from('<f',src,i+4*k)[0];j=i+16+4*k;idx=struct.unpack_from('<I',src,j)[0]
            if not math.isfinite(w) or not 0<=w<=1.001:raise ValueError('异常权重')
            if w:
                if idx>=len(REMAP):raise ValueError('顶点组索引越界')
                struct.pack_into('<I',dst,j,REMAP[idx]);count+=1
    if not count or dst==src:raise ValueError('无有效映射或已迁移')
    return bytes(dst),count

def migrate(d):
    if d.path.name.lower()!='ningguang.ini':return d.data,[],[]
    text=d.data.decode();hashes=[m[2].lower() for m in HASHLINE.finditer(text)]
    if not any(h in PAIRS for h in hashes):return d.data,[],[]
    s=Section('Ningguang historical migration',0,len(d.lines))
    def fail(msg):return d.data,[decision(d,s,'MIG-001','needs_diagnosis',[],msg)],[]
    if any(new in hashes or hashes.count(old)!=n for old,(new,n) in PAIRS.items()):return fail('旧 Hash 计数不符或新旧混用')
    for part in ('head','body','dress'):
        sec=d.by.get('textureoverrideningguang'+part)
        if not sec:return fail('缺少 '+part)
        for slot,kind in ((0,'diffuse'),(1,'lightmap')):
            ref='resourceningguang'+part+kind
            if d.value(sec,f'ps-t{slot}').lower()!=ref or not d.file(ref):return fail('布局或资源不符 '+part)
    bp=d.file('ResourceNingguangBlend')
    if not bp:return fail('Blend 缺失或路径不安全')
    try:old=bp.read_bytes();new,n=remap_blend(old)
    except (OSError,ValueError) as e:return fail(str(e))
    out=HASHLINE.sub(lambda m:m[1]+PAIRS[m[2].lower()][0]+m[3] if m[2].lower() in PAIRS else m[0],text).encode()
    return out,[decision(d,s,'MIG-001','auto_repair',['五组 Hash 精确计数','Head/Body/Dress 布局核验','Blend 32 字节记录','119 项映射',f'有效权重索引 {n}'],patch=out)],[(bp,old,new)]
