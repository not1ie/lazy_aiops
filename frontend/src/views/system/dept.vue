<template>
  <el-card>
    <template #header>
      <div class="flex justify-between items-center">
        <span class="font-bold">部门管理</span>
        <el-button type="primary" icon="Plus" @click="handleAdd">新增部门</el-button>
      </div>
    </template>

    <el-table
      :data="tableData"
      style="width: 100%"
      row-key="id"
      default-expand-all
      :tree-props="{ children: 'children' }"
    >
      <el-table-column prop="name" label="部门名称" />
      <el-table-column prop="leader" label="负责人" width="120" />
      <el-table-column prop="phone" label="联系电话" width="150" />
      <el-table-column prop="sort" label="排序" width="80" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <el-button size="small" type="primary" link @click="handleEdit(row)">编辑</el-button>
          <el-button size="small" type="primary" link @click="handleAddSub(row)">新增下级</el-button>
          <el-button size="small" type="danger" link @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="visible" :title="isEdit ? '编辑部门' : '新增部门'" width="520px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="上级部门">
          <el-tree-select
            v-model="form.parent_id"
            :data="tableData"
            :props="{ label: 'name', value: 'id' }"
            check-strictly
            clearable
            placeholder="选择上级部门"
          />
        </el-form-item>
        <el-form-item label="部门名称" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="负责人">
          <el-input v-model="form.leader" />
        </el-form-item>
        <el-form-item label="联系电话">
          <el-input v-model="form.phone" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="form.email" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :label="1">启用</el-radio>
            <el-radio :label="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="submit">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const tableData = ref([])
const visible = ref(false)
const isEdit = ref(false)
const form = reactive({
  id: '',
  name: '',
  parent_id: '',
  sort: 0,
  leader: '',
  phone: '',
  email: '',
  status: 1
})
const authHeaders = () => ({
  Authorization: 'Bearer ' + localStorage.getItem('token')
})

const resetForm = () => {
  form.id = ''
  form.name = ''
  form.parent_id = ''
  form.sort = 0
  form.leader = ''
  form.phone = ''
  form.email = ''
  form.status = 1
}

const fetchData = async () => {
  const res = await axios.get('/api/v1/system/depts', { headers: authHeaders() })
  if (res.data.code === 0) tableData.value = res.data.data
}

const handleAdd = () => {
  isEdit.value = false
  resetForm()
  visible.value = true
}

const handleAddSub = (row) => {
  isEdit.value = false
  resetForm()
  form.parent_id = row.id
  visible.value = true
}

const handleEdit = (row) => {
  isEdit.value = true
  form.id = row.id || ''
  form.name = row.name || ''
  form.parent_id = row.parent_id || ''
  form.sort = Number(row.sort || 0)
  form.leader = row.leader || ''
  form.phone = row.phone || ''
  form.email = row.email || ''
  form.status = Number(row.status ?? 1)
  visible.value = true
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除部门 "${row.name}" 吗？`, '删除确认', { type: 'warning' })
    await axios.delete(`/api/v1/system/depts/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    fetchData()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

const submit = async () => {
  try {
    const payload = {
      name: form.name?.trim(),
      parent_id: form.parent_id || '',
      sort: Number(form.sort || 0),
      leader: form.leader?.trim() || '',
      phone: form.phone?.trim() || '',
      email: form.email?.trim() || '',
      status: Number(form.status ?? 1)
    }
    if (!payload.name) {
      ElMessage.warning('部门名称不能为空')
      return
    }
    if (isEdit.value) {
      await axios.put(`/api/v1/system/depts/${form.id}`, payload, { headers: authHeaders() })
    } else {
      await axios.post('/api/v1/system/depts', payload, { headers: authHeaders() })
    }
    ElMessage.success('操作成功')
    visible.value = false
    fetchData()
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

onMounted(fetchData)
</script>
