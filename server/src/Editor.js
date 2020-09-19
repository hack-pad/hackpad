import CodeMirror from 'codemirror/lib/codemirror';
import 'codemirror/lib/codemirror.css';
import 'codemirror/theme/material-darker.css';
import 'codemirror/mode/go/go';

import { listenColorScheme } from './ColorScheme';
import './Editor.css';

export function newEditor(elem, onEdit) {
  const editor = CodeMirror(elem, {
    mode: "go",
    theme: "default",
    lineNumbers: true,
    indentUnit: 4,
    indentWithTabs: true,
  })
  listenColorScheme({
    light: () => editor.setOption("theme", "default"),
    dark: () => editor.setOption("theme", "material-darker"),
  })
  editor.on('change', onEdit)
  return {
    getContents() {
      return editor.getValue()
    },

    setContents(contents) {
      editor.setValue(contents)
    },

    getCursorIndex() {
      return editor.getCursor().ch
    },

    setCursorIndex(index) {
      editor.setCursor({ ch: index })
    },
  }
}
