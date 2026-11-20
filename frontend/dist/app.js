/* ORFix Repair V4.0 – React UI. No JSX/transpiler required in shipped bundle.
   Wails calls go.main.App; a browser falls back to explicit simulation mode. */
(function () {
  'use strict';
  const h = React.createElement;
  const Glass = window.ORFixLiquidGlass;
  const demo = !(window.go && window.go.main && window.go.main.App);
  const bridge = demo ? null : window.go.main.App;
  const image = (key) => './assets/' + key + '.png';
  const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));

  function Icon({name, size=18, stroke=1.85}){
    const common={width:size,height:size,viewBox:'0 0 24 24',fill:'none',stroke:'currentColor',strokeWidth:stroke,strokeLinecap:'round',strokeLinejoin:'round','aria-hidden':true};
    const paths={
      folder: [h('path',{d:'M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z'})],
      chevron: [h('path',{d:'m9 18 6-6-6-6'})],
      arrow: [h('path',{d:'M5 12h14m-6-6 6 6-6 6'})],
      shield: [h('path',{d:'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z'}),h('path',{d:'m9 12 2 2 4-4'})],
      check: [h('path',{d:'m4 12 5 5L20 6'})],
      refresh: [h('path',{d:'M20 7v5h-5M4 17v-5h5'}),h('path',{d:'M5 9a8 8 0 0 1 14-2M19 15a8 8 0 0 1-14 2'})],
      eye: [h('path',{d:'M2 12s3.5-6.5 10-6.5 10 6.5 10 6.5S18.5 18.5 12 18.5 2 12Z'}),h('circle',{cx:12,cy:12,r:2.5})],
      log: [h('path',{d:'M6 4h12v16H6zM9 9h6M9 13h6M9 17h4'})],
      info: [h('circle',{cx:12,cy:12,r:9}),h('path',{d:'M12 11v5M12 7.5h.01'})],
      alert: [h('path',{d:'m12 3 10 18H2L12 3ZM12 9v5M12 18h.01'})],
      cross: [h('path',{d:'M5 5l14 14M19 5 5 19'})],
      caret: [h('path',{d:'m6 9 6 6 6-6'})],
      sparkle:[h('path',{d:'M12 3l2 6 6 2-6 2-2 6-2-6-6-2 6-2 2-6ZM19 17l.7 2.3L22 20l-2.3.7L19 23l-.7-2.3L16 20l2.3-.7L19 17Z'})],
      close:[h('path',{d:'M18 6 6 18M6 6l12 12'})],
    };
    return h('svg',common,...(paths[name]||paths.info));
  }

  function GlassButton({children,kind='ghost',onClick,disabled=false,small=false,leading,iconOnly=false}){
    return h('button',{className:'glass-button '+kind+(small?' compact':'')+(iconOnly?' icon-only':''),onClick,disabled},
      h(Glass,{className:'button-optic',radius:16,blur:10},h('span',{className:'button-label'},leading&&h(Icon,{name:leading,size:18}),children)));
  }

  const viewConfig={
    idle:{kicker:'准备就绪',title:'让 Mod 恢复正常显示',desc:'选择 Mod 文件夹，一键检查并修复已知的材质兼容问题。',image:null,tone:'neutral'},
    working:{kicker:'正在处理',title:'正在检查 Mod 文件…',desc:'正在分析 INI 与已验证的兼容规则，请勿关闭窗口。',image:null,tone:'neutral'},
    success:{kicker:'修复完成',title:'已完成正常修复',desc:'未检测到需要进一步确认的面部绑定风险。',image:'success',tone:'good'},
    risk:{kicker:'需要你确认',title:'疑似存在面部兼容问题',desc:'普通材质修复已完成。请进入游戏查看面部是否有白圈或妆容消失。',image:'face-risk',tone:'warn'},
    faceWorking:{kicker:'正在处理',title:'正在修复面部绑定…',desc:'将仅修改符合条件的面部 Diffuse 绑定。',image:null,tone:'neutral'},
    faceSuccess:{kicker:'修复完成',title:'面部兼容修复已完成',desc:'请返回游戏检查面部显示，所有修改已建立备份。',image:'face-done',tone:'good'},
    failed:{kicker:'出现错误',title:'本次修复未完全成功',desc:'查看下方详细日志，可利用备份恢复已修改的文件。',image:'failure',tone:'bad'},
    faceFailed:{kicker:'修复未完成',title:'面部修复遇到问题',desc:'面部修改可能未全部完成，请查看日志后再决定是否回滚。',image:'failure',tone:'bad'},
    preview:{kicker:'扫描结束',title:'预览完成，未修改文件',desc:'本次仅列出修复候选及潜在风险，可以安全关闭。',image:null,tone:'neutral'},
    rollback:{kicker:'操作完成',title:'已检查最近批次备份',desc:'回滚数量及文件详情已记录在日志中。',image:null,tone:'good'}
  };

  function simulatedResult(kind,folder){
    const root=folder||'C:\\Games\\GIMI\\Mods';
    if(kind==='preview')return {phase:'仅预览（未改动文件）',root,scanned:8,modified:0,faceRisks:['RaidenShogun: FaceHeadDiffuse 使用旧版绑定'],changes:['预览：RaidenShogun.ini（3 项材质调整）','预览：Ningguang.ini（兼容修复候选）'],errors:[],logPath:''};
    if(kind==='rollback')return {phase:'恢复最近一次修复',root,scanned:8,modified:2,faceRisks:[],changes:['已恢复两份 INI/BUF 备份'],errors:[],logPath:''};
    if(kind==='failed')return {phase:'正常增强修复',root,scanned:8,modified:1,faceRisks:[],changes:['已备份 RaidenShogun.ini'],errors:['Ningguang.ini：资源文件不可写'],logPath:''};
    if(kind==='success')return {phase:'正常增强修复',root,scanned:8,modified:3,faceRisks:[],changes:['已修复 RaidenShogun.ini','已修复 Ningguang.ini','已修复 Ineffa.ini'],errors:[],logPath:''};
    if(kind==='face')return {phase:'用户确认后的 Face Fix',root,scanned:1,modified:1,faceRisks:[],changes:['已完成 RaidenShogun FaceHeadDiffuse：ps-t0 → this','已保存备份及回滚清单'],errors:[],logPath:''};
    return {phase:'正常增强修复',root,scanned:8,modified:3,faceRisks:['RaidenShogun.ini：FaceHeadDiffuse 存在旧式 ps-t0 绑定'],changes:['已修复 RaidenShogun.ini（保留 TexFx）','已修复 Ningguang.ini（Hash + Blend + NNFix）','已修复 Ineffa.ini（EyeHead）'],errors:[],logPath:''};
  }

  class App extends React.Component {
    constructor(props){
      super(props);
      this.state={folder:demo?'C:\\Games\\GIMI\\Mods':'',phase:'idle',result:null,logsOpen:false,scenario:'risk',error:'',busy:false};
    }
    componentDidMount(){if(bridge&&bridge.DefaultFolder)bridge.DefaultFolder().then(x=>x&&this.setState({folder:x})).catch(()=>{});}
    async selectFolder(){
      const {busy,phase,folder}=this.state;
      if(busy||phase==='risk')return;
      if(demo){const value=prompt('交互演示：请输入展示用 Mod 目录（不会访问文件）',folder);if(value)this.setState({folder:value});return;}
      try{const val=await bridge.SelectDirectory();if(val)this.setState({folder:val});}catch(e){this.setState({error:'选择目录失败：'+String(e)});}
    }
    async execute(action){
      const {busy,folder,scenario}=this.state;
      if(busy)return;
      if(!folder){this.setState({error:'请先选择 Mod 文件夹'});return;}
      this.setState({error:'',busy:true,phase:'working',logsOpen:false});
      try{
        let res;
        if(demo){await sleep(650);res=simulatedResult(action==='normal'?scenario:action,folder);}
        else res=await bridge[action==='normal'?'RunNormal':action==='preview'?'RunPreview':'RunRollback'](folder);
        const errors=res.errors||res.Errors||[];
        const risks=res.faceRisks||res.FaceRisks||[];
        this.setState({result:res,phase:errors.length?'failed':action==='preview'?'preview':action==='rollback'?'rollback':risks.length?'risk':'success'});
      }catch(e){this.setState({error:String(e),result:{errors:[String(e)]},phase:'failed'});}
      finally{this.setState({busy:false});}
    }
    async continueFace(){
      const {busy,phase,folder}=this.state;
      if(phase!=='risk'||busy)return;
      this.setState({busy:true,phase:'faceWorking'});
      try{
        let res;if(demo){await sleep(620);res=simulatedResult('face',folder);}else res=await bridge.ContinueFace();
        const errs=res.errors||res.Errors||[];
        this.setState({result:res,phase:errs.length?'faceFailed':'faceSuccess'});
      }catch(e){this.setState({result:{errors:[String(e)]},phase:'faceFailed'});}finally{this.setState({busy:false});}
    }
    async finish(){
      if(this.state.busy)return;
      if(!demo && bridge && bridge.Finish) {try{await bridge.Finish();}catch(e){this.setState({error:String(e)});}}
      this.setState({phase:'idle',result:null,logsOpen:false});
    }
    render(){
      const {folder,phase,result,logsOpen,scenario,error,busy}=this.state;
    const cfg=viewConfig[phase]||viewConfig.idle;
    const risks=result?(result.faceRisks||result.FaceRisks||[]):[];
    const changes=result?(result.changes||result.Changes||[]):[];
    const errors=result?(result.errors||result.Errors||[]):[];
    const count=result?(result.modified||result.Modified||0):0;
    const scanned=result?(result.scanned||result.Scanned||0):0;
    const showButtons=phase!=='risk'&&phase!=='faceWorking';
    const outputLog=changes.concat(errors.map(x=>'错误：'+x)).concat(risks.map(x=>'风险：'+x));
    const stage=phase==='idle'?0:phase==='working'?1:phase==='risk'?2:(phase==='faceWorking'||phase==='faceSuccess'||phase==='faceFailed')?3:1;

    return h('main',{className:'app-canvas'},
      h('div',{className:'app-glow', 'aria-hidden':true}),
      h('section',{className:'app-shell'},
        h('header',{className:'app-header'},
          h('div',{className:'brand'},
            h('img',{src:image('logo'),alt:'ORFix',className:'brand-logo'}),
            h('div',{className:'brand-text'},h('strong',null,'ORFix Repair'),h('small',null,'V4.1.0'))),
          demo ? h('details',{className:'demo-options header-demo'},h('summary',null,'切换演示结果'),h('div',{className:'demo-picks'},
            [['success','普通成功'],['risk','疑似面部风险'],['failed','修复失败']].map(([v,lab])=>h('button',{key:v,className:scenario===v?'selected':'',onClick:()=>this.setState({scenario:v,phase:'idle',result:null,logsOpen:false})},lab)))) : null
        ),
        h('div',{className:'content'+(logsOpen?' logs-mode':'')},
          h('div',{className:'section-header'},h('h1',null,'修复 Mod 贴图与兼容问题')),
          h('div',{className:'folder-row'},
            h('div',{className:'folder-field'},h(Icon,{name:'folder',size:19}),h('span',{title:folder||'未选择'},folder||'选择你的 GIMI / Mods 文件夹')),
            h(GlassButton,{onClick:()=>this.selectFolder(),disabled:busy||phase==='risk',small:true,leading:'folder'},'浏览目录')
          ),
          h('div',{className:'actions',style:{display:showButtons?'flex':'none'}},
            h(GlassButton,{kind:'primary',onClick:()=>this.execute('normal'),disabled:busy,leading:busy?'refresh':'sparkle'},busy?'正在修复…':'开始正常修复'),
            h(GlassButton,{kind:'secondary',onClick:()=>this.execute('preview'),disabled:busy,leading:'eye'},'仅预览'),
            h(GlassButton,{kind:'secondary',onClick:()=>this.execute('rollback'),disabled:busy,leading:'refresh'},'回滚最近批次')
          ),
          h('div',{className:'flow-line'},
            h('span',{className:stage>=1?'flow-node active':'flow-node'},h('b',null,'1'),'常规修复'),
            h('span',{className:'flow-connector'},''),
            h('span',{className:stage>=2?'flow-node active':'flow-node'},h('b',null,'2'),'游戏中检查'),
            h('span',{className:'flow-connector'},''),
            h('span',{className:stage>=3?'flow-node active':'flow-node'},h('b',null,'3'),'可选 Face Fix')
          ),
          h(Glass,{className:'result-panel',radius:23,blur:14},
            h('div',{className:'result-inner '+cfg.tone},
              h('div',{className:'result-copy'},
                h('div',{className:'status-kicker'},h('span',{className:'status-indicator'}),cfg.kicker),
                h('h2',null,cfg.title),
                h('p',null,cfg.desc),
                phase==='risk'?h('div',{className:'risk-actions'},
                  h('button',{className:'risk-primary',onClick:()=>this.continueFace(),disabled:busy},'继续修复 Face Fix',h(Icon,{name:'arrow',size:17})),
                  h('button',{className:'risk-done',onClick:()=>this.finish(),disabled:busy},'完成退出')
                ):phase!=='idle'&&phase!=='working'&&phase!=='faceWorking'?h('div',{className:'result-numbers'},
                  h('span',null,h('strong',null,scanned),' 个 INI 已扫描'),
                  h('span',{className:'result-sep'},''),
                  h('span',null,h('strong',null,count),' 个文件已处理'),
                  h('button',{onClick:()=>this.finish(),className:'mini-done'},'完成退出'))
                :null
              ),
              cfg.image?h('img',{className:'state-illustration',alt:cfg.title,src:image(cfg.image)}):h('div',{className:'idle-art','aria-hidden':true},h('div',{className:'idle-circles'},h(Icon,{name:phase==='working'?'refresh':'sparkle',size:34,stroke:1.4})))
            )
          ),
          h('div',{className:'bottom-line'},
            h('button',{className:'log-head','aria-controls':'orfix-log-panel','aria-expanded':logsOpen,onClick:()=>this.setState({logsOpen:!logsOpen})},h(Icon,{name:'log',size:17}),'详细日志',h('span',{className:'log-secondary'},outputLog.length?outputLog.length+' 条记录':'修复过程与备份记录'),h('span',{className:'log-arrow '+(logsOpen?'expanded':'')},h(Icon,{name:'caret',size:16})))
          ),
          logsOpen&&h('div',{className:'log-panel',id:'orfix-log-panel',role:'log','aria-label':'详细修复日志',tabIndex:0},outputLog.length?outputLog.map((x,i)=>h('div',{className:errors.includes(x.replace('错误：',''))?'log-bad':'log-row',key:i},x)):h('div',{className:'log-muted'},'暂无日志。开始扫描后将在此显示结果。')),
          error&&h('div',{className:'inline-error'},h(Icon,{name:'alert',size:16}),error)
        ),
        null
      )
    );
    }
  }
  ReactDOM.render(h(App),document.getElementById('root'));
})();
