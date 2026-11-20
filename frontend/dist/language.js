/* ORFix Repair: bilingual production UI layer.
   Localization is applied while React creates its elements. This deliberately
   avoids MutationObserver DOM text rewrites, which race with React state changes. */
(function () {
  'use strict';
  const dict = {
    '准备就绪':'Ready',
    '让 Mod 恢复正常显示':'Get Your Mods Working Again',
    '选择 Mod 文件夹，一键检查并修复已知的材质兼容问题。':'Select a Mod folder to scan and repair known material compatibility issues.',
    '正在处理':'Working',
    '正在修复面部绑定…':'Repairing facial bindings…',
    '正在检查 Mod 文件…':'Scanning Mod files…',
    '正在分析 INI 与已验证的兼容规则，请勿关闭窗口。':'Analyzing INI files and verified compatibility rules. Please do not close the window.',
    '修复完成':'Repair Complete',
    '已完成正常修复':'Normal repair completed',
    '未检测到需要进一步确认的面部绑定风险。':'No facial binding issues requiring further confirmation were detected.',
    '需要你确认':'Action Required',
    '疑似存在面部兼容问题':'Possible facial compatibility issue detected',
    '普通材质修复已完成。请进入游戏查看面部是否有白圈或妆容消失。':'Normal material repair is complete. Check in-game for white circles or missing makeup.',
    '将仅修改符合条件的面部 Diffuse 绑定。':'Only eligible facial Diffuse bindings will be modified.',
    '面部兼容修复已完成':'Facial compatibility repair completed',
    '请返回游戏检查面部显示，所有修改已建立备份。':'Return to the game and check the face. Backups were created for all changes.',
    '出现错误':'Error',
    '本次修复未完全成功':'Repair was not fully successful',
    '查看下方详细日志，可利用备份恢复已修改的文件。':'Check the detailed logs below. Backups can be used to restore modified files.',
    '修复未完成':'Repair Incomplete',
    '面部修复遇到问题':'Face repair encountered a problem',
    '面部修改可能未全部完成，请查看日志后再决定是否回滚。':'The face changes may be incomplete. Review the logs before deciding whether to roll back.',
    '扫描结束':'Scan Complete',
    '预览完成，未修改文件':'Preview complete — no files modified',
    '本次仅列出修复候选及潜在风险，可以安全关闭。':'This preview only lists potential fixes and risks. It is safe to close.',
    '操作完成':'Operation Complete',
    '已检查最近批次备份':'Latest backup batch checked',
    '回滚数量及文件详情已记录在日志中。':'Rollback counts and file details have been recorded in the logs.',
    '切换演示结果':'Demo Scenarios',
    '普通成功':'Repair Success',
    '疑似面部风险':'Possible Face Issue',
    '修复失败':'Repair Failed',
    '修复 Mod 贴图与兼容问题':'Fix Mod Textures & Compatibility Issues',
    '未选择':'Not selected',
    '选择你的 GIMI / Mods 文件夹':'Select your GIMI / Mods folder',
    '浏览目录':'Browse Folder',
    '正在修复…':'Repairing…',
    '开始正常修复':'Start Normal Repair',
    '仅预览':'Preview Only',
    '回滚最近批次':'Rollback Latest Batch',
    '常规修复':'Normal Repair',
    '游戏中检查':'In-game Check',
    '可选 Face Fix':'Optional Face Fix',
    '继续修复 Face Fix':'Continue Face Fix',
    '完成退出':'Finish',
    '个 INI 已扫描':'INI files scanned',
    '个文件已处理':'files processed',
    '详细日志':'Detailed Logs',
    '详细修复日志':'Detailed repair logs',
    '修复过程与备份记录':'Repair and backup activity',
    '暂无日志。开始扫描后将在此显示结果。':'No logs yet. Results will appear here after scanning.',
    '请先选择 Mod 文件夹':'Please select a Mod folder first.',
    '交互演示：请输入展示用 Mod 目录（不会访问文件）':'Demo only: enter a sample Mod folder path (no files will be accessed).',
    '预览：RaidenShogun.ini（3 项材质调整）':'Preview: RaidenShogun.ini (3 material adjustments)',
    '预览：Ningguang.ini（兼容修复候选）':'Preview: Ningguang.ini (compatibility fix candidate)',
    '已恢复两份 INI/BUF 备份':'Restored two INI/BUF backups',
    '已备份 RaidenShogun.ini':'Backed up RaidenShogun.ini',
    'Ningguang.ini：资源文件不可写':'Ningguang.ini: resource file is not writable',
    '已修复 RaidenShogun.ini':'Repaired RaidenShogun.ini',
    '已修复 Ningguang.ini':'Repaired Ningguang.ini',
    '已修复 Ineffa.ini':'Repaired Ineffa.ini',
    '已完成 RaidenShogun FaceHeadDiffuse：ps-t0 → this':'Updated RaidenShogun FaceHeadDiffuse: ps-t0 → this',
    '已保存备份及回滚清单':'Saved backup and rollback manifest',
    '已修复 RaidenShogun.ini（保留 TexFx）':'Repaired RaidenShogun.ini (TexFx preserved)',
    '已修复 Ningguang.ini（Hash + Blend + NNFix）':'Repaired Ningguang.ini (Hash + Blend + NNFix)',
    '已修复 Ineffa.ini（EyeHead）':'Repaired Ineffa.ini (EyeHead)',
    'RaidenShogun: FaceHeadDiffuse 使用旧版绑定':'RaidenShogun: FaceHeadDiffuse uses a legacy binding',
    'RaidenShogun.ini：FaceHeadDiffuse 存在旧式 ps-t0 绑定':'RaidenShogun.ini: FaceHeadDiffuse contains a legacy ps-t0 binding',
    '错误：':'Error: ',
    '风险：':'Risk: '
  };

  let language = 'zh';
  try { language = localStorage.getItem('orfix-language') === 'en' ? 'en' : 'zh'; } catch (_) {}

  // Preserve filesystem paths verbatim. A valid folder name may contain CJK characters;
  // those characters must never be mistaken for untranslated diagnostic text.
  function isFileSystemPath(value) {
    const candidate = String(value || '').trim();
    return /^(?:[A-Za-z]:[\\/]|\\\\|\/|\.{1,2}[\\/])/.test(candidate)
      || /^file:\/\//i.test(candidate);
  }

  function translateText(value) {
    if (typeof value !== 'string' || language !== 'en') return value;
    // Preserve whitespace around individual React text nodes (e.g. count labels).
    const match = value.match(/^(\s*)([\s\S]*?)(\s*)$/);
    if (!match) return value;
    const [, left, core, right] = match;
    if (!core || core === '中文') return value;
    // Paths are data, not UI copy: keep the complete path even when it contains Chinese characters.
    if (isFileSystemPath(core)) return value;
    if (Object.prototype.hasOwnProperty.call(dict, core)) return left + dict[core] + right;
    if (!/[\u3400-\u9fff]/.test(core)) return value;
    let m = core.match(/^(\d+) 条记录$/);
    if (m) return left + m[1] + ' entries' + right;
    m = core.match(/^(\d+) 个 INI 已扫描$/);
    if (m) return left + m[1] + ' INI files scanned' + right;
    m = core.match(/^(\d+) 个文件已处理$/);
    if (m) return left + m[1] + ' files processed' + right;
    // Dynamic results: translate known prefixes and preserve technical data.
    if (core.startsWith('错误：')) return left + 'Error: ' + translateText(core.slice(3)) + right;
    if (core.startsWith('风险：')) return left + 'Risk: ' + translateText(core.slice(3)) + right;
    if (core.startsWith('选择目录失败：')) return left + 'Failed to select a folder: ' + core.slice(7) + right;
    // Actual Go repair results are generated at runtime, not static JSX strings.
    // Translate known diagnostics while preserving technical file names, hashes and paths.
    let detail = core;
    const clauses = [
      ['[凝光历史迁移]', '[Ningguang legacy migration]'],
      ['[TexFx 专项]', '[TexFx special case]'],
      ['[实验 TexFx]', '[Experimental TexFx]'],
      ['[Face Fix 人工检查]', '[Face Fix manual review]'],
      ['修复后又被其他操作修改', 'changed by another operation after repair'],
      ['扫描范围内未发现 ORFix 全局库', 'The ORFix global library was not found in the selected scan range'],
      ['扫描范围内未发现 TexFx 全局库', 'The TexFx global library was not found in the selected scan range'],
      ['namespace = global\\ORFix 或 ORFix.ini', 'namespace = global\\ORFix or ORFix.ini'],
      ['若全局库安装在扫描目录之外，请忽略', 'Ignore this warning if the global library is installed outside the scanned folder'],
      ['否则 NNFix/ORFix 调用可能无法生效', 'otherwise NNFix/ORFix calls may not take effect'],
      ['若 Mod 使用 TexFx，请确认已安装对应版本', 'If the Mod uses TexFx, ensure the appropriate version is installed'],
      ['若库位于扫描目录之外，请忽略', 'Ignore if the library is outside the scanned folder'],
      ['同时启用多个替换同一角色的 Mod 可能冲突，建议游戏测试时只启用一个。', 'Enabling multiple replacement Mods for one character may cause conflicts; test with only one active.'],
      ['个凝光版本配置', 'Ningguang configurations'],
      ['发现', 'Found'],
      ['发现旧式 Reflection ShaderRegex/CommandList', 'Legacy Reflection ShaderRegex/CommandList detected'],
      ['含旧式 Reflection ShaderRegex/CommandList', 'Contains legacy Reflection ShaderRegex/CommandList'],
      ['需检查是否与 ORFix 冲突', 'check for possible ORFix conflicts'],
      ['为防冲突，未自动改写', 'not modified automatically to avoid conflicts'],
      ['疑似独立眼睛/头发子网格，但 IB 文件未核实', 'Possible separate eye/hair submesh, but the IB file was not verified'],
      ['独立眼睛网格', 'Separate eye mesh'],
      ['独立头发网格', 'Separate hair mesh'],
      ['自动修复会跳过', 'automatic repair will skip it'],
      ['双贴图 Diffuse + LightMap，已存在 NNFix', 'Two-texture Diffuse + LightMap with an existing NNFix call'],
      ['双贴图 Diffuse + LightMap，缺少 NNFix', 'Two-texture Diffuse + LightMap missing an NNFix call'],
      ['三贴图 NormalMap + Diffuse + LightMap，已存在 ORFix', 'Three-texture NormalMap + Diffuse + LightMap with an existing ORFix call'],
      ['三贴图 NormalMap + Diffuse + LightMap，缺少 ORFix', 'Three-texture NormalMap + Diffuse + LightMap missing an ORFix call'],
      ['满足资源校验，可作为 NNFix 修复候选', 'resource checks passed; eligible NNFix candidate'],
      ['满足资源校验，可作为 ORFix 修复候选', 'resource checks passed; eligible ORFix candidate'],
      ['若仍发绿，需进一步检查 Shader、倒影或贴图实际语义', 'if still green, inspect shaders, reflection and texture semantics'],
      ['贴图槽位未能严格分类', 'Texture slots cannot be reliably classified'],
      ['不推断应使用 ORFix 还是 NNFix', 'cannot determine whether to use ORFix or NNFix'],
      ['检测到 ps-t69 + TexFx 透明遮罩', 'ps-t69 + TexFx transparency mask detected'],
      ['检测到透明相关配置', 'Transparency-related configuration detected'],
      ['透明网格需按 IB/UV/绘制范围分析，不能仅靠全局 NNFix 推断', 'Transparent meshes require IB/UV/draw-range analysis; global NNFix alone is insufficient'],
      ['INI 含二进制 NUL 数据', 'INI contains binary NUL data'],
      ['孤立 Head/非常规名称，可能是武器或道具，安全起见跳过', 'Isolated Head or unusual name may be a weapon/accessory; skipped for safety'],
      ['识别为独立眼睛/头发子网格，但 IB 资源文件无法核实', 'Separate eye/hair submesh detected, but IB resource file could not be verified'],
      ['存在透明调用及复杂网格/条件绘制，须手动检查', 'Transparency call with complex mesh or conditional drawing requires manual review'],
      ['未知透明材质不自动修改', 'unknown transparent materials are not automatically modified'],
      ['含 if/else 等条件分支，暂不自动修复', 'Contains if/else branches; automatic repair skipped'],
      ['额外槽位和透明命令需按网格检查', 'Additional slots and transparency calls need per-mesh checking'],
      ['额外/复杂贴图槽位', 'Additional or complex texture slots'],
      ['已识别固定混合半透明 CustomShaderTransparency', 'Recognized fixed blended-transparency CustomShaderTransparency'],
      ['在其执行前插入 NNFix 并保留透明绘制', 'inserted NNFix before it and preserved transparency drawing'],
      ['双贴图部件已接入 NNFix', 'Two-texture part now uses NNFix'],
      ['双贴图存在不确定性', 'Two-texture layout is ambiguous'],
      ['资源缺失、其他 CommandList 或命名不对应', 'missing resources, other CommandLists, or mismatched names'],
      ['双贴图不完整或已有修复', 'Two-texture layout incomplete or already fixed'],
      ['找不到三张贴图资源定义或实际文件', 'Could not locate definitions or files for all three textures'],
      ['贴图名称或槽位冲突', 'Texture names or slots conflict'],
      ['ps-t0/ps-t1 均缺失，不能排除作者有意留空', 'Both ps-t0 and ps-t1 are absent; author may have left them blank intentionally'],
      ['恢复', 'Restored'],
      ['个被注释的贴图槽位', 'commented-out texture slots'],
      ['补充', 'Added'],
      ['个缺失的贴图槽位', 'missing texture slots'],
      ['接入 ORFix', 'Enabled ORFix'],
      ['文件过大，跳过', 'File too large; skipped'],
      ['分支内贴图已匹配，绘制前插入', 'Branch texture layout matched; inserted before drawing'],
      ['贴图布局不明确', 'Texture layout is ambiguous'],
      ['透明配置不满足严格自动修复条件', 'Transparency configuration does not meet strict auto-fix criteria'],
      ['保留原样', 'left unchanged'],
      ['已验证双贴图和透明遮罩，在透明渲染前添加 NNFix', 'Verified two textures and transparency mask; added NNFix before transparency rendering'],
      ['保留 ps-t69 及 Transparency 原版本', 'kept ps-t69 and the original Transparency version'],
      ['同一 Face 区块同时存在 ps-t0 与 this，可能存在分支差异', 'A Face section contains both ps-t0 and this; branches may differ'],
      ['不是可确认的面部 Diffuse 资源绑定', 'is not a verified facial Diffuse resource binding'],
      ['引用的面部贴图资源不存在或文件缺失', 'references a missing facial texture resource or file'],
      ['资源不是可识别的贴图文件', 'resource is not a recognized texture file'],
      ['面部区块直接绑定 ps-t0，但其调用链另有 this 赋值', 'Face section directly uses ps-t0, but its call chain also assigns this'],
      ['同时被非面部区块调用，不能安全地全局修改', 'also used by non-face sections; global changes are unsafe'],
      ['同时存在 ps-t0 与 this，需人工检查分支', 'contains both ps-t0 and this; branches need manual inspection'],
      ['含有非面部 Diffuse 的 ps-t0，不能自动修改', 'uses non-facial Diffuse ps-t0; cannot auto-fix'],
      ['引用的面部贴图文件缺失', 'referenced facial texture file is missing'],
      ['的资源不是可识别贴图文件', 'resource is not a recognized texture file'],
      ['无法解析面部调用链', 'Cannot resolve Face call chain'],
      ['没有可安全自动修改的 ps-t0 行', 'No safely editable ps-t0 lines'],
      ['疑似面部兼容风险，但调用链存在歧义', 'Possible facial compatibility risk; call chain is ambiguous'],
      ['疑似面部兼容风险', 'Possible facial compatibility risk'],
      ['保留原配置，需人工检查', 'Original configuration preserved; manual review required'],
      ['转换失败', 'conversion failed'],
      ['发现面部 Diffuse 使用 ps-t0 的疑似风险', 'Possible risk: facial Diffuse uses ps-t0'],
      ['可选 Face Fix 候选', 'optional Face Fix candidate'],
      ['此项不是游戏故障确认', 'this is not confirmation of an in-game issue'],
      ['将面部专属 Diffuse 绑定从 ps-t0 改为 this', 'Changed facial Diffuse binding from ps-t0 to this'],
      ['疑似面部问题', 'Possible facial issue'],
      ['可安全自动修改', 'safely auto-fixable'],
      ['旧 Hash', 'old Hash'],
      ['新旧 Hash 混用', 'old/new Hash values mixed'],
      ['避免部分替换', 'avoided partial replacement'],
      ['应出现', 'should occur'],
      ['实际', 'actual'],
      ['不满足已验证结构', 'does not match the verified structure'],
      ['贴图布局未知', 'unknown texture layout'],
      ['贴图缺失', 'texture missing'],
      ['找不到', 'could not find'],
      ['缺少 Blend 文件定义', 'Blend resource definition missing'],
      ['Blend 路径不安全', 'Unsafe Blend path'],
      ['读取 Blend 失败', 'Failed to read Blend'],
      ['Blend 可能已重映射，拒绝重复处理', 'Blend may already be remapped; duplicate processing rejected'],
      ['Hash 替换计数异常', 'Hash replacement count unexpected'],
      ['活动顶点组索引', 'active vertex-group index'],
      ['超出映射表', 'out of remap bounds'],
      ['Blend.buf 未发现有效骨骼权重', 'No valid bone weights found in Blend.buf'],
      ['顶点组映射表长度错误', 'Invalid vertex-group remap length'],
      ['偏移', 'offset'],
      ['存在异常权重', 'contains an invalid weight'],
      ['Blend.buf 必须是非空的', 'Blend.buf must contain non-empty'],
      ['字节记录', 'byte records'],
      ['按官方 119 项映射迁移', 'remapped using the 119-entry reference'],
      ['按 119 项映射迁移', 'remapped using 119 entries'],
      ['个非零权重索引', 'nonzero weight indices'],
      ['历史 Hash 五组已核实更新', 'five verified legacy Hash groups updated'],
      ['Blend 119 项映射已应用', '119-entry Blend mapping applied'],
      ['共', 'total'],
      ['个权重索引', 'weight indices'],
      ['分段透明渲染', 'segmented transparent rendering'],
      ['没有足够可靠的注释定位', 'cannot reliably identify a mesh from comments'],
      ['没有可安全自动修改的面部绑定，复杂情况需要人工检查', 'No safely editable face binding found; complex cases require manual inspection'],
      ['请先完成当前面部检查，再更换目录', 'Finish the current face check before changing folders'],
      ['请先完成待确认的面部阶段', 'Finish the pending face stage first'],
      ['请先完成当前面部检查', 'Finish the current face check first'],
      ['面部检查未结束，不能回滚其他目录', 'Face check is still pending; cannot roll back another folder'],
      ['当前没有等待用户确认的面部风险', 'No pending facial risk awaiting user confirmation'],
      ['Wails 窗口尚未初始化', 'Wails window is not initialized'],
      ['扫描后文件发生变化', 'File changed after scanning'],
      ['不是普通文件', 'Not a regular file'],
      ['文件为只读', 'File is read-only'],
      ['创建备份失败', 'Failed to create backup'],
      ['写入备份失败', 'Failed to write backup'],
      ['备份文件名冲突过多', 'Too many backup filename collisions'],
      ['无法创建临时文件', 'Failed to create temporary file'],
      ['写入临时文件失败', 'Failed to write temporary file'],
      ['替换文件失败', 'Failed to replace file'],
      ['原文件未被删除，备份保留', 'original file preserved; backup retained'],
      ['预检查读取失败', 'Pre-check read failed'],
      ['预检查 Hash 不一致', 'Pre-check Hash mismatch'],
      ['目标不是可写普通文件', 'Target is not a writable regular file'],
      ['同一文件被重复列入修复计划', 'The same file appears twice in the repair plan'],
      ['回退前 Hash 不匹配', 'Pre-rollback Hash mismatch'],
      ['自动回退部分失败', 'Partial automatic rollback failure'],
      ['已将此前成功写入的文件自动回退', 'Previously modified files were rolled back automatically'],
      ['回滚清单写入失败', 'Failed to write rollback manifest'],
      ['日志写入失败', 'Failed to write log'],
      ['跨阶段文件 Hash 不连续', 'Cross-stage file Hash continuity failure'],
      ['无法合并两阶段回滚记录', 'Failed to merge rollback records across stages'],
      ['保存回滚清单失败', 'Failed to save rollback manifest'],
      ['当前会话完整回滚清单', 'Full session rollback manifest'],
      ['回滚清单', 'Rollback manifest'],
      ['备份', 'Backup'],
      ['已修复', 'Repaired'],
      ['已完成 Face Fix', 'Completed Face Fix'],
      ['失败', 'Failed'],
      ['警告', 'Warning'],
      ['成功', 'Success'],
      ['待修复', 'Pending fix'],
      ['读取最近一次修复记录失败', 'Failed to read latest repair record'],
      ['最近一次修复记录格式无效', 'Latest repair record format is invalid'],
      ['跳过不安全路径', 'Skipped unsafe path'],
      ['读原文件失败', 'Failed to read original file'],
      ['备份丢失', 'Backup missing'],
      ['备份哈希校验失败', 'Backup Hash verification failed'],
      ['回滚失败', 'Rollback failed'],
      ['已恢复', 'Restored'],
      ['回滚前状态备份', 'Pre-rollback state backup'],
      ['不安全的相对路径', 'Unsafe relative path'],
      ['路径超出扫描根目录', 'Path escapes scan root'],
      ['文件修复后又被其他操作修改', 'File was changed after repair by another operation'],
      ['资源文件不可写', 'Resource file is not writable'],
      [' 行。', ' lines.'],
      [' 行', ' lines'],
      ['项', 'items'],
      ['条', 'entries'],
      ['行', 'lines'],
      ['跳过', 'Skipped'],
      ['文件太大', 'File too large'],
      ['已跳过', 'Skipped'],
      ['未自动修改', 'Not automatically modified'],
      ['跳过自动修改', 'Skipped automatic modification'],
      ['未修改', 'Unchanged'],
      ['需人工检查', 'Manual review required'],
      ['人工检查', 'Manual review'],
      ['请检查', 'Please check'],
      ['面部', 'facial'],
      ['目录', 'directory'],
      ['文件', 'file'],
      ['错误', 'error'],
      ['第 ', 'line '],
      [' 行', ' line'],
      [' 个', ' '],
      [' 项', ' items'],
      [' 行。', ' lines.'],
      ['，', ', '],
      ['；', '; '],
      ['：', ': '],
      ['（', '('],
      ['）', ')'],
      ['。', '.'],
    ];
    // Longest phrases are applied first; short fragments only clean up residual
    // diagnostic messages that include filenames, resource identifiers and counts.
    clauses.sort((a,b) => b[0].length - a[0].length);
    for (const [from,to] of clauses) detail = detail.split(from).join(to);
    detail = detail.replace(/\s+/g, ' ').trim();
    if (/[\u3400-\u9fff]/.test(detail)) {
      // Never silently display a misleading partial translation in EN mode.
      // The original detailed diagnostics remain in the saved engine log.
      const names = detail.match(/[\w.\\/:-]+\.(?:ini|buf|dds|json)/ig) || [];
      return left + (names.length ? names.slice(0,3).join(', ') + ' — ' : '')
        + 'Additional diagnostic needs manual review (see saved log).' + right;
    }
    return left + detail + right;
  }

  function localizeProps(props) {
    if (!props) return props;
    let next = props;
    for (const key of ['title', 'aria-label', 'placeholder', 'alt']) {
      if (typeof props[key] === 'string') {
        const translated = translateText(props[key]);
        if (translated !== props[key]) {
          if (next === props) next = Object.assign({}, props);
          next[key] = translated;
        }
      }
    }
    return next;
  }

  function localizeChild(v) {
    if (typeof v === 'string') return translateText(v);
    if (Array.isArray(v)) return v.map(localizeChild);
    return v;
  }

  function createElement(type, props, ...children) {
    return React.createElement(type, localizeProps(props), ...children.map(localizeChild));
  }

  function setLanguage(next) {
    const wanted = next === 'en' ? 'en' : 'zh';
    if (language === wanted) return;
    language = wanted;
    try { localStorage.setItem('orfix-language', language); } catch (_) {}
    syncDocument();
    window.dispatchEvent(new Event('orfix-language-changed'));
  }

  function syncDocument() {
    document.documentElement.lang = language === 'en' ? 'en' : 'zh-CN';
    document.title = language === 'en' ? 'ORFix Repair · English V4.1.2' : 'ORFix Repair · 中文 V4.1.2';
  }
  syncDocument();

  const originalPrompt = window.prompt;
  window.prompt = function (message, value) {
    return originalPrompt.call(window, translateText(message), value);
  };

  
  window.ORFixI18n = { setLanguage, getLanguage: () => language, createElement, translateText };
})();
