<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="login-title">{{ isRegister ? '注册账号' : '99点单' }}</h1>

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
          <div class="input-with-button">
            <input
              v-model="form.email"
              type="email"
              class="form-input"
              placeholder="请输入邮箱"
              required
            />
            <button
              type="button"
              class="btn btn-secondary send-code-btn"
              @click="handleSendCode"
              :disabled="codeSending || countdown > 0"
            >
              {{ countdown > 0 ? `${countdown}s` : (codeSending ? '发送中...' : '发送验证码') }}
            </button>
          </div>
        </div>

        <div v-if="isRegister" class="form-group">
          <label class="form-label">验证码</label>
          <input
            v-model="form.code"
            type="text"
            class="form-input"
            placeholder="请输入邮箱验证码"
            maxlength="6"
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
        <a href="#" @click.prevent="toggleMode">
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
const codeSending = ref(false)
const countdown = ref(0)
const errorMsg = ref('')
const successMsg = ref('')

const form = reactive({
  username: '',
  password: '',
  email: '',
  code: ''
})

let countdownTimer = null

function toggleMode() {
  isRegister.value = !isRegister.value
  errorMsg.value = ''
  successMsg.value = ''
  form.code = ''
}

async function handleSendCode() {
  if (!form.email) {
    errorMsg.value = '请先输入邮箱'
    return
  }

  // 简单的邮箱格式验证
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(form.email)) {
    errorMsg.value = '请输入有效的邮箱地址'
    return
  }

  errorMsg.value = ''
  codeSending.value = true

  try {
    await authStore.sendVerificationCode(form.email)
    successMsg.value = '验证码已发送到您的邮箱'

    // 开始倒计时
    countdown.value = 60
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(countdownTimer)
      }
    }, 1000)
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    codeSending.value = false
  }
}

async function handleSubmit() {
  errorMsg.value = ''
  successMsg.value = ''
  loading.value = true

  try {
    if (isRegister.value) {
      if (!form.code) {
        errorMsg.value = '请输入验证码'
        loading.value = false
        return
      }
      await authStore.register(form.username, form.password, form.email, form.code)
      successMsg.value = '注册成功，请登录'
      isRegister.value = false
      form.password = ''
      form.code = ''
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

<style scoped>
.input-with-button {
  display: flex;
  gap: 10px;
}

.input-with-button .form-input {
  flex: 1;
}

.send-code-btn {
  white-space: nowrap;
  min-width: 100px;
  padding: 8px 12px;
  font-size: 14px;
}

.send-code-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
