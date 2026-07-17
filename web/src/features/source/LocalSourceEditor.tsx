import Editor, { loader } from '@monaco-editor/react'
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api'
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import 'monaco-editor/esm/vs/basic-languages/go/go.contribution'
import 'monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution'
import 'monaco-editor/esm/vs/basic-languages/python/python.contribution'
import 'monaco-editor/esm/vs/basic-languages/typescript/typescript.contribution'

self.MonacoEnvironment = {
  getWorker: () => new EditorWorker(),
}
loader.config({ monaco })

export default function LocalSourceEditor({
  language,
  value,
}: {
  language: string
  value: string
}) {
  return (
    <Editor
      height="100%"
      language={language}
      loading={<div className="source-loading">Loading local source…</div>}
      options={{
        readOnly: true,
        minimap: { enabled: false },
        fontSize: 12,
        lineNumbersMinChars: 3,
        scrollBeyondLastLine: false,
        wordWrap: 'off',
        renderLineHighlight: 'line',
        padding: { top: 12 },
      }}
      theme="vs-dark"
      value={value}
    />
  )
}
