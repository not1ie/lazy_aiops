<template>
  <div class="apple-table-wrapper">
    <table class="apple-table">
      <thead>
        <tr>
          <th v-for="column in columns" :key="column.key" :style="{ width: column.width }">
            {{ column.title }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, index) in data" :key="index" @click="handleRowClick(row)">
          <td v-for="column in columns" :key="column.key">
            <slot :name="`cell-${column.key}`" :row="row" :value="row[column.key]">
              {{ row[column.key] }}
            </slot>
          </td>
        </tr>
        <tr v-if="!data || data.length === 0">
          <td :colspan="columns.length" class="empty-state">
            <i class="fas fa-inbox"></i>
            <p>暂无数据</p>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
const props = defineProps({
  columns: {
    type: Array,
    required: true
  },
  data: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['row-click'])

const handleRowClick = (row) => {
  emit('row-click', row)
}
</script>

<style scoped>
.apple-table-wrapper {
  overflow-x: auto;
  border-radius: var(--radius-lg);
  background: var(--apple-bg-secondary);
}

.apple-table {
  width: 100%;
  border-collapse: collapse;
}

thead {
  background: var(--apple-bg-tertiary);
}

th {
  padding: var(--space-md);
  text-align: left;
  font-weight: 600;
  color: var(--apple-text-secondary);
  font-size: 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

td {
  padding: var(--space-md);
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  color: var(--apple-text-primary);
  font-size: 14px;
}

tbody tr {
  transition: background 0.2s var(--ease-standard);
  cursor: pointer;
}

tbody tr:hover {
  background: var(--apple-bg-tertiary);
}

.empty-state {
  text-align: center;
  padding: var(--space-2xl) !important;
  color: var(--apple-text-tertiary);
}

.empty-state i {
  font-size: 48px;
  margin-bottom: var(--space-md);
  opacity: 0.3;
}

.empty-state p {
  margin: 0;
  font-size: 14px;
}
</style>
