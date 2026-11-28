from pathlib import Path
import sys
from PySide6.QtCore import QObject,Signal,Slot,Property,QSettings,QTimer
from PySide6.QtWidgets import QFileDialog,QApplication
from engine.service import run,Result

class Controller(QObject):
    changed=Signal()
    def __init__(self):
        super().__init__();self.settings=QSettings();self._language=self.settings.value('language','zh');self._folder=str(Path(sys.executable if getattr(sys,'frozen',False) else sys.argv[0]).resolve().parent);self._phase='idle';self._busy=False;self._r=None;self._prior=[];self._face_root=None
    @Property(str,notify=changed)
    def language(self):return self._language
    @Property(str,notify=changed)
    def folder(self):return self._folder
    @Property(str,notify=changed)
    def phase(self):return self._phase
    @Property(bool,notify=changed)
    def busy(self):return self._busy
    @Property(int,notify=changed)
    def scanned(self):return self._r.scanned if self._r else 0
    @Property(int,notify=changed)
    def modified(self):return self._r.modified if self._r else 0
    @Property(str,notify=changed)
    def logPath(self):return self._r.log_path if self._r else ''
    @Property(str,notify=changed)
    def logText(self):
        if not self._r:return ''
        r=self._r
        if self._language=='zh':return '\n'.join(r.changes+['风险：'+x for x in r.face_risks]+['提示：'+x for x in r.warnings]+['错误：'+x for x in r.errors]+[f'{d.rule_id} | {d.section} | {d.decision}' for d in r.records if d.decision in ('needs_diagnosis','blocked','superseded')]+(['日志：'+r.log_path] if r.log_path else []))
        # Technical paths, section names and hashes are never translated.
        labels={'auto_repair':'Repair candidate','requires_confirmation':'Face confirmation required','needs_diagnosis':'Manual diagnosis required','no_change':'No change','blocked':'Protected / skipped','superseded':'Later rule matched'}
        return '\n'.join([f'{labels.get(d.decision,d.decision)} | {d.rule_id} | {d.section} | {d.execution}' for d in r.records]+[f'Processed: {r.modified}; scanned: {r.scanned}']+(['Warnings: see original saved log.'] if r.warnings else [])+(['Errors: see original saved log.'] if r.errors else [])+(['Detailed log: '+r.log_path] if r.log_path else []))
    @Slot(str)
    def setLanguage(self,v):
        if v in ('zh','en'):self._language=v;self.settings.setValue('language',v);self.changed.emit()
    @Slot()
    def browse(self):
        if self._busy or self._phase=='risk':return
        folder=QFileDialog.getExistingDirectory(None,'选择 Mod 文件夹' if self._language=='zh' else 'Select Mod folder',self._folder)
        if folder:self._folder=folder;self.changed.emit()
    @Slot(str)
    def execute(self,action):
        if self._busy or action not in ('normal','preview','rollback','face'):return
        if self._phase=='risk' and action!='face':return
        if action=='face' and (self._phase!='risk' or self._face_root!=self._folder):return
        self._busy=True;self._phase='faceWorking' if action=='face' else 'working';self.changed.emit()
        QTimer.singleShot(60,lambda:self._run(action))
    def _run(self,action):
        try:
            r=run(self._folder,action,self._prior if action=='face' else None);self._r=r
            if action=='normal':self._prior=r.entries;self._face_root=self._folder
            if r.errors or not r.scanned and action!='rollback':self._phase='faceFailed' if action=='face' else 'failed'
            elif action=='normal' and r.face_risks:self._phase='risk'
            elif action=='face':self._phase='faceSuccess' if r.modified else 'faceFailed'
            else:self._phase={'normal':'success','preview':'preview','rollback':'rollback'}[action]
        except Exception as e:self._r=Result(action,errors=[str(e)]);self._phase='failed'
        finally:self._busy=False;self.changed.emit()
    @Slot()
    def finish(self):
        if not self._busy:QApplication.quit()
