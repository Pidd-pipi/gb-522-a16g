import axios from 'axios'
import { ElMessage } from 'element-plus'

export const api = axios.create({ baseURL: '/api/v1', timeout: 30000 })

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('otdr_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  config.headers['X-Request-ID'] = crypto.randomUUID()
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const code = error.response?.data?.error?.code ?? 'NETWORK_ERROR'
    const message = error.response?.data?.error?.message ?? '请求未完成'
    ElMessage.error(`${message} [${code}]`)
    if (error.response?.status === 401) {
      localStorage.removeItem('otdr_token')
      localStorage.removeItem('otdr_user')
      if (location.pathname !== '/login') location.assign('/login')
    }
    return Promise.reject(error)
  },
)
