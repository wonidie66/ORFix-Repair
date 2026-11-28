from dataclasses import dataclass, asdict, field
import json, difflib
from pathlib import Path
RULES={
'MAT-001':('标准双槽位 NNFix','已知 Diffuse/LightMap 输入约定，补充 NNFix。'),
'MAT-002':('标准三槽位 ORFix','已知 NormalMap/Diffuse/LightMap 输入约定，恢复缺失绑定或补充 ORFix。'),
'MAT-003':('四槽位 NNFix','已验证 Diffuse/LightMap/MetalMap/ShadowRamp 布局。'),
'MAT-004':('复杂静态材质 ORFix','核实三贴图语义及绘制路径，保留辅助槽位；必要时纠正顺序。'),
'MAT-006':('动态材质 ORFix','验证 IB/VB 关联及每个互斥材质路径，补充带条件保护的 ORFix。'),
'MAT-007':('CommandList 材质分析','追踪本地命令调用及独立绑定组，避免跨分支拼接。'),
'TEX-001':('安全 TexFx 兼容','已知双贴图与单一透明遮罩调用，保留遮罩及绘制。'),
'FACE-001':('面部 Diffuse','使用 this 针对匹配资源替换，避免固定槽位依赖。'),
'FACE-002+003':('多变体面部与 LightMap','伊涅芙 B 组合修复实测有效；停用额外固定绑定，单项效果未隔离。'),
'MIG-001':('凝光历史迁移','严格五组 Hash 签名与 119 项 Blend 映射联动。'),
'TEX-RECIPE-001':('茜特菈莉专项','只接受历史精确 SHA-256 和必需资源。'),
'SAFE-001':('安全检查','证据不足或配置不支持，保留原文件。')}
@dataclass
class Decision:
    rule_id:str; section:str; decision:str; checks:list=field(default_factory=list); reason:str=''; target_hash:str=''; diff:list=field(default_factory=list); file:str=''; execution:str='not_written'; backup:str=''
    def record(self):
        name,mechanism=RULES[self.rule_id]
        components=['FACE-002','FACE-003'] if self.rule_id=='FACE-002+003' else [self.rule_id]
        if '辅助 Outline 槽位：True' in self.checks:components.append('MAT-005')
        if any(x.startswith('局部语法修正：') and not x.endswith('：0') for x in self.checks):components.append('INI-001')
        return dict(asdict(self),rule_components=components,rule_name=name,mechanism=mechanism,evidence='V4.2.1 历史源码及归档规则；候选静态分析',validation_limit='未验证当前运行时 Shader 与游戏画面',invariants='除规则明确列出的修改外，保留其他资源、配置和绘制逻辑')

def delta(a,b):return list(difflib.unified_diff(a.decode('utf-8',errors='replace').splitlines(),b.decode('utf-8',errors='replace').splitlines(),fromfile='before',tofile='after',lineterm='',n=1))
def decision(d,s,r,status,checks,reason='',patch=None):
    return Decision(r,s.name,status,checks,reason,d.value(s,'hash'),delta(d.data,patch) if patch else [],str(d.path))
def write_reports(root,stamp,phase,records,summary):
    root=Path(root);base=root/f'ORFixRepair_{stamp}_{phase}'
    data={'version':'5.0.0','phase':phase,'summary':summary,'decisions':[r.record() for r in records]}
    base.with_suffix('.json').write_text(json.dumps(data,ensure_ascii=False,indent=2),encoding='utf-8')
    lines=['ORFix Repair V5.0.0',phase,json.dumps(summary,ensure_ascii=False)]
    for r in data['decisions']:
        lines.extend(['━'*36,f"规则编号：{r['rule_id']} | {r['rule_name']}",f"【目标定位】{r['file']} | {r['section']} | Hash: {r['target_hash']}",f"【判断依据】{'；'.join(r['checks'])}",f"【修复原理】{r['mechanism']}",f"【决策】{r['decision']} | {r['reason']}",'【候选修改】',*r['diff'],f"【保护范围】{r['invariants']}",f"【规则依据】{r['evidence']}",f"【验证边界】{r['validation_limit']}",f"【执行状态】{r['execution']} | {r['backup']}"])
    base.with_suffix('.log').write_text('\n'.join(lines)+'\n',encoding='utf-8')
    return str(base.with_suffix('.log'))
