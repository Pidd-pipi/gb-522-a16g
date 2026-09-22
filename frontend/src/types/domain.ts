import type { EventMarker } from './event'

export interface FiberRoute { id: number; route_code: string; name: string; length_m: number; refractive_index: number; launch_connector: string; route_status: 'active' | 'maintenance' | 'retired'; baseline_trace_id?: number; created_at: string; updated_at: string }
export interface BlockingCase { id: number; case_status: string; current_trace_id: number; created_at: string }
export interface PendingCase { id: number; baseline_trace_id: number; current_trace_id: number; case_status: string; conclusion: string; created_at: string }
export type BaselineRejectReason = 'open_cases' | 'trace_not_on_route' | 'same_baseline' | 'baseline_concurrent_change'
export interface BaselineRejection { id: number; requested_trace_id: number; current_trace_id?: number; result: 'rejected' | 'succeeded'; reason: BaselineRejectReason | 'success'; actor_name: string; blocking_case_ids: number[]; created_at: string }
export interface RouteDetail { route: FiberRoute; traces: TraceCapture[]; pending_cases: PendingCase[]; rejections: BaselineRejection[] }
export interface TraceCapture { id: number; route_id: number; wavelength_nm: number; pulse_width_ns: number; sample_interval_ns: number; raw_points_json: number[]; processed_points_json: number[]; noise_floor_db: number; denoise_window: number; peak_threshold_db: number; merge_window: number; captured_at: string; uploaded_by: number; created_at: string }
export interface TraceDetail { id: number; route_id: number; wavelength_nm: number; pulse_width_ns: number; sample_interval_ns: number; points: number[]; processed_points: number[]; noise_floor_db: number; denoise_window: number; peak_threshold_db: number; merge_window: number; captured_at: string; uploaded_by: number }
export interface TraceEnvelope { trace: TraceDetail; events: EventMarker[] }
export interface User { id: number; username: string; display_name: string; role: 'analyst' | 'reviewer' | 'admin' }
export interface AuditLog { id: number; actor_id: number; actor_name: string; action: string; resource_type: string; resource_id: number; route_id?: number; request_id: string; before: string; after: string; created_at: string }
export interface PageMeta { page: number; page_size: number; total: number }
export interface ApiEnvelope<T> { data: T; meta?: PageMeta; request_id: string }
