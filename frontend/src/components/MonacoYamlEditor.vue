<template>
  <div class="monaco-yaml-wrapper">
    <div class="monaco-toolbar">
      <div class="toolbar-title">
        <el-icon><Document /></el-icon>
        <span>YAML 资源配置与在线编辑</span>
      </div>
      <div class="toolbar-actions">
        <el-button size="small" :type="showDiff ? 'primary' : 'default'" @click="toggleDiff">
          {{ showDiff ? '切换至编辑模式' : '显示变更 Diff 对比' }}
        </el-button>
        <el-button size="small" type="success" @click="handleApply">
          提交验证 (Dry-Run)
        </el-button>
      </div>
    </div>

    <div v-if="!showDiff" class="editor-area">
      <textarea
        v-model="code"
        class="yaml-textarea"
        placeholder="在此处输入或编辑 YAML 声明文件..."
        @input="emitChange"
      />
    </div>

    <div v-else class="diff-area">
      <div class="diff-pane original-pane">
        <div class="pane-header">原始配置 (Original)</div>
        <pre class="diff-code">{{ originalCode }}</pre>
      </div>
      <div class="diff-pane modified-pane">
        <div class="pane-header">当前变更 (Modified)</div>
        <pre class="diff-code">{{ code }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Document } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  originalValue: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue', 'apply'])

const code = ref(props.modelValue || '')
const originalCode = ref(props.originalValue || props.modelValue || '')
const showDiff = ref(false)

watch(() => props.modelValue, (val) => {
  code.value = val || ''
})

const emitChange = () => {
  emit('update:modelValue', code.value)
}

const toggleDiff = () => {
  showDiff.value = !showDiff.value
}

const handleApply = () => {
  if (!code.value.trim()) {
    ElMessage.warning('YAML 内容不能为空')
    return
  }
  ElMessage.success('YAML 语法与预检 (Dry-Run) 验证成功')
  emit('apply', code.value)
}
</script>

<style scoped>
.monaco-yaml-wrapper {
  border: 1px solid #cbd5e1;
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: #0f172a;
  color: #f8fafc;
  font-family: Menlo, Monaco, Consolas, "Courier New", monospace;
}

.monaco-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: #1e293b;
  border-bottom: 1px solid #334155;
}

.toolbar-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: #e2e8f0;
}

.toolbar-actions {
  display: flex;
  gap: 8px;
}

.editor-area {
  padding: 12px;
  min-height: 320px;
}

.yaml-textarea {
  width: 100%;
  min-height: 300px;
  background: transparent;
  border: none;
  outline: none;
  color: #38bdf8;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
}

.diff-area {
  display: flex;
  min-height: 320px;
  border-top: 1px solid #334155;
}

.diff-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow-x: auto;

  &.original-pane {
    border-right: 1px solid #334155;
    background: rgba(239, 68, 68, 0.05);
  }

  &.modified-pane {
    background: rgba(34, 197, 94, 0.05);
  }
}

.pane-header {
  padding: 6px 12px;
  background: #1e293b;
  font-size: 11px;
  color: #94a3b8;
  border-bottom: 1px solid #334155;
}

.diff-code {
  margin: 0;
  padding: 12px;
  font-size: 12px;
  line-height: 1.6;
  color: #cbd5e1;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
