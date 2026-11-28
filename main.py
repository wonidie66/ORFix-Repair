"""V5 native desktop entry; command-line core mode also works without Qt."""
import argparse,json,sys,os
from pathlib import Path
from dataclasses import asdict
from engine.service import run
BASE=Path(__file__).resolve().parent

def cli(args):
    if args.action=='face' and not args.confirm_face:raise SystemExit('Face Fix requires --confirm-face after inspecting game rendering.')
    prior=None
    if args.action=='face' and args.merge_latest:
        from engine.transaction import LATEST
        manifest=json.loads((Path(args.directory)/LATEST).read_bytes());prior=manifest['items']
    r=run(args.directory,args.action,prior)
    print(json.dumps(asdict(r),ensure_ascii=False,indent=2,default=str));return 1 if r.errors else 0

def main():
    p=argparse.ArgumentParser(description='ORFix Repair V5.0.0 native desktop / CLI')
    p.add_argument('action',nargs='?',choices=['preview','normal','face','rollback']);p.add_argument('directory',nargs='?');p.add_argument('--confirm-face',action='store_true');p.add_argument('--merge-latest',action='store_true');p.add_argument('--self-test-output');p.add_argument('--safe-ui',action='store_true');a=p.parse_args()
    if a.action:
        if not a.directory:p.error('directory is required')
        return cli(a)
    if a.self_test_output:
        os.environ.setdefault('QT_QPA_PLATFORM','offscreen');os.environ.setdefault('QT_QUICK_BACKEND','software')
    from PySide6.QtCore import QUrl,QTimer
    from PySide6.QtGui import QIcon
    from PySide6.QtWidgets import QApplication
    from PySide6.QtQml import QQmlApplicationEngine
    from PySide6.QtQuick import QQuickWindow
    from PySide6.QtQuickControls2 import QQuickStyle
    from ui.controller import Controller
    app=QApplication(sys.argv[:1]);app.setOrganizationName('ORFixRepair');app.setApplicationName('ORFix Repair');app.setWindowIcon(QIcon(str(BASE/'ui/assets/icon.ico')));QQuickStyle.setStyle('Basic')
    controller=Controller();engine=QQmlApplicationEngine();engine.rootContext().setContextProperty('backend',controller)
    shader = None
    if not a.safe_ui and os.environ.get('QT_QUICK_BACKEND','').lower() != 'software' and not os.environ.get('ORFIX_SAFE_UI'):
        from ui.shader_build import prepare_shader
        try:
            shader = prepare_shader()
        except (OSError, RuntimeError) as exc:
            print(f'Liquid Glass fallback: {exc}',file=sys.stderr)
    if shader:
        print('Liquid Glass QSB ready; actual GPU rendering still depends on Qt/driver.')
    engine.rootContext().setContextProperty('glassShaderReady', bool(shader))
    engine.rootContext().setContextProperty('glassShaderUrl', QUrl.fromLocalFile(str(shader)) if shader else QUrl())
    warnings=[]
    engine.warnings.connect(lambda items: warnings.extend(x.toString() for x in items))
    for qmlpath in (BASE/'PySide6/qml', BASE/'PySide6/Qt/qml'):
        if qmlpath.is_dir():engine.addImportPath(str(qmlpath))
    engine.load(QUrl.fromLocalFile(str(BASE/'ui/Main.qml')))
    if not engine.rootObjects():
        if a.self_test_output:Path(a.self_test_output).write_text('{"qml_loaded": false}',encoding='utf8')
        return 2
    if a.self_test_output:
        def done():
            bad=[x for x in warnings if any(t in x for t in ('ReferenceError','TypeError','Cannot assign','is not a type','not installed','Unable to assign'))]
            Path(a.self_test_output).write_text(json.dumps({'qml_loaded':not bad,'window':engine.rootObjects()[0].objectName(),'warnings':warnings},ensure_ascii=False),encoding='utf8');app.exit(2 if bad else 0)
        QTimer.singleShot(400,done)
    return app.exec()
if __name__=='__main__':sys.exit(main())
