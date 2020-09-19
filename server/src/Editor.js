import CodeMirror from 'codemirror/lib/codemirror';
import 'codemirror/lib/codemirror.css';
import 'codemirror/theme/material-darker.css';
import 'codemirror/mode/go/go';

import './Editor.css';

export function newEditor(elem, onEdit) {
  const editor = CodeMirror(elem, {
    mode: "go",
    theme: "default",
    lineNumbers: true,
    indentUnit: 4,
    indentWithTabs: true,
  })
  if (window.matchMedia) {
    const setTheme = mq => {
      editor.setOption("theme", mq.matches ? "material-darker" : "default")
    }
    const media = window.matchMedia("(prefers-color-scheme: dark)")
    setTheme(media)
    media.addListener(setTheme)
  }
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
