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
    viewportMargin: Infinity,
  })
  listenColorScheme({
    light: () => editor.setOption("theme", "default"),
    dark: () => editor.setOption("theme", "material-darker"),
  })
  editor.on('change', onEdit)

  elem.addEventListener('click', e => {
    editor.focus()
    if (e.target === elem) {
      // If we've clicked outside the code editor area, then it must be the bottom empty space.
      editor.setCursor({ line: editor.lineCount()-1 })
    }
  })
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
