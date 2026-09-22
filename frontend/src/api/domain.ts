import { api } from './client'
import type { ApiEnvelope, AuditLog, FiberRoute, TraceCapture, TraceEnvelope, User } from '@/types/domain'
import type { EventMarker, EventType } from '@/types/event'
import type { LocalizationCase } from '@/types/case'

export const authApi = { login: (body: { username: string; password: string }) => api.post<ApiEnvelope<{ token: string; expires_at: string; user: User }>>('/auth/login', body) }
export const routeApi = {
  list: (params?: object) => api.get<ApiEnvelope<FiberRoute[]>>('/routes', { params }),
  create: (body: object) => api.post<ApiEnvelope<FiberRoute>>('/routes', body),
  update: (id: number, body: object) => api.patch<ApiEnvelope<FiberRoute>>(`/routes/${id}`, body),
  detail: (id: number) => api.get<ApiEnvelope<{ route: FiberRoute; traces: TraceCapture[] }>>(`/routes/${id}`),
  setBaseline: (id: number, traceId: number) => api.post<ApiEnvelope<FiberRoute>>(`/routes/${id}/baseline`, { trace_id: traceId }),
}
export const traceApi = {
  list: (params?: object) => api.get<ApiEnvelope<TraceCapture[]>>('/traces', { params }),
  detail: (id: number) => api.get<ApiEnvelope<TraceEnvelope>>(`/traces/${id}`),
  import: (body: object) => api.post<ApiEnvelope<TraceCapture>>('/traces/import', body),
  detect: (id: number, body: object) => api.post<ApiEnvelope<{ trace_id: number; detected_count: number; noise_floor_db: number; threshold_db: number; rejected_out_of_bounds: number }>>(`/traces/${id}/detect`, body),
}
export const eventApi = {
  list: (params?: object) => api.get<ApiEnvelope<EventMarker[]>>('/events', { params }),
  review: (id: number, body: { event_type: EventType; distance_m?: number; review_note: string }) => api.patch<ApiEnvelope<EventMarker>>(`/events/${id}/review`, body),
}
export const caseApi = {
  list: (params?: object) => api.get<ApiEnvelope<LocalizationCase[]>>('/cases', { params }),
  create: (body: object) => api.post<ApiEnvelope<LocalizationCase>>('/cases', body),
  detail: (id: number) => api.get<ApiEnvelope<{ case: LocalizationCase; differences: object[] }>>(`/cases/${id}`),
  analyze: (id: number, body: object) => api.post<ApiEnvelope<LocalizationCase>>(`/cases/${id}/analyze`, body),
  confirm: (id: number, body: object) => api.post<ApiEnvelope<LocalizationCase>>(`/cases/${id}/confirm`, body),
  close: (id: number, version: number) => api.post<ApiEnvelope<LocalizationCase>>(`/cases/${id}/close`, { version }),
}
export const auditApi = { list: (params?: object) => api.get<ApiEnvelope<AuditLog[]>>('/audit', { params }) }
