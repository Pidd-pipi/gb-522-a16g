export const EVENT_TYPES = ['connector', 'splice', 'bend', 'break', 'end', 'unknown'] as const
export type EventType = (typeof EVENT_TYPES)[number]

export interface EventMarker {
  id: number
  trace_id: number
  distance_m: number
  event_type: EventType
  insertion_loss_db: number
  reflectance_db: number
  confidence: number
  algorithm_event_type: EventType
  algorithm_distance_m: number
  algorithm_insertion_loss_db: number
  reviewed: boolean
  review_note: string
  reviewed_by?: number
  reviewed_at?: string
  created_at: string
}

export const eventLabel: Record<EventType, string> = {
  connector: '连接器', splice: '熔接点', bend: '弯曲', break: '断点', end: '光纤末端', unknown: '未知',
}
