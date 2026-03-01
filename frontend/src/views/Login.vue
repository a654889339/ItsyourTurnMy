<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="login-title">{{ isRegister ? '注册账号' : '财务管理系统' }}</h1>

      <div v-if="errorMsg" class="message message-error">{{ errorMsg }}</div>
      <div v-if="successMsg" class="message message-success">{{ successMsg }}</div>

      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label class="form-label">用户名</label>
          <input
            v-model="form.username"
            type="text"
            class="form-input"
            placeholder="请输入用户名"
            required
          />
        </div>

        <div v-if="isRegister" class="form-group">
          <label class="form-label">邮箱</label>
          <input
            v-model="form.email"
            type="email"
            class="form-input"
            placeholder="请输入邮箱"
            required
          />
        </div>

        <div class="form-group">
          <label class="form-label">密码</label>
          <input
            v-model="form.password"
            type="password"
            class="form-input"
            placeholder="请输入密码"
            required
          />
        </div>

        <button type="submit" class="btn btn-primary" style="width: 100%; margin-top: 10px" :disabled="loading">
          {{ loading ? '处理中...' : (isRegister ? '注册' : '登录') }}
        </button>
      </form>

      <p style="text-align: center; margin-top: 20px; color: #666">
        {{ isRegister ? '已有账号？' : '没有账号？' }}
        <a href="#" @click.prevent="isRegister = !isRegister">
          {{ isRegister ? '立即登录' : '立即注册' }}
        </a>
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../store/auth'

const router = useRouter()
const authStore = useAuthStore()

const isRegister = ref(false)
const loading = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

const form = reactive({
  username: '',
  password: '',
  email: ''
})

async function handleSubmit() {
  errorMsg.value = ''
  successMsg.value = ''
  loading.value = true

  try {
    if (isRegister.value) {
      await authStore.register(form.username, form.password, form.email)
      successMsg.value = '注册成功，请登录'
      isRegister.value = false
      form.password = ''
    } else {
      await authStore.login(form.username, form.password)
      router.push('/')
    }
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    loading.value = false
  }
}
</script>
