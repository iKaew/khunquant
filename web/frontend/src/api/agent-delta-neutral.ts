export interface DeltaNeutralPlanListItem {
  id: number
  name: string
  asset: string
  status: string
  mode: string
  spot_provider: string
  spot_account: string
  spot_symbol: string
  futures_provider: string
  futures_account: string
  futures_symbol: string
  capital_usdt: number
  monitor_interval: string
  enabled: boolean
  cross_exchange: boolean
  health_score: number
  health_label: string
  last_checked_at?: string
  last_alert_at?: string
  fee_snapshot?: {
    trading_fee_usdt: number
    funding_fee_usdt: number
    fetched_at: string
  } | null
  created_at: string
  updated_at: string
}

export interface DeltaNeutralMonitorSnapshot {
  id: number
  plan_id: number
  checked_at: string
  spot_price: number
  spot_quantity: number
  spot_value_usdt: number
  futures_mark_price: number
  futures_contracts: number
  futures_notional_usdt: number
  futures_unrealized_pnl_usdt: number
  current_funding_rate: number
  estimated_next_funding_usdt: number
  funding_state: string
  delta_drift_pct: number
  liquidation_price: number
  liquidation_distance_pct: number
  margin_ratio_pct: number
  margin_state: string
  health_score: number
  health_label: string
  cross_exchange: boolean
  threshold_breached: boolean
  breach_codes: string[]
  data_status: string
  error_msg?: string
  agent_invoked: boolean
  created_at: string
}

export interface DeltaNeutralAlert {
  id: number
  plan_id: number
  snapshot_id?: number
  triggered_at: string
  severity: string
  code: string
  message: string
  recommended_action?: string
  agent_invoked: boolean
  delivered_channel?: string
  delivered_chat_id?: string
  created_at: string
}

export interface DeltaNeutralExecutionLeg {
  id: number
  execution_id: number
  leg_type: string
  provider: string
  account?: string
  symbol: string
  side: string
  order_type: string
  requested_amount: number
  requested_notional_usdt: number
  requested_price: number
  order_id?: string
  state: string
  filled_quantity: number
  filled_notional_usdt: number
  avg_fill_price: number
  fee_usdt: number
  error_msg?: string
  created_at: string
  updated_at: string
}

export interface DeltaNeutralExecution {
  id: number
  plan_id: number
  attempt_id: string
  state: string
  requested_at: string
  approved_at?: string
  completed_at?: string
  error_msg?: string
  legs: DeltaNeutralExecutionLeg[]
  created_at: string
  updated_at: string
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    let message = `API error: ${res.status} ${res.statusText}`
    try {
      const text = await res.text()
      if (text.trim()) message = text.trim()
    } catch {
      // ignore
    }
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

export interface ListDeltaNeutralPlansParams {
  enabled?: boolean
  status?: string
}

export async function listDeltaNeutralPlans(
  params?: ListDeltaNeutralPlansParams,
): Promise<DeltaNeutralPlanListItem[]> {
  const q = new URLSearchParams()
  if (params?.enabled !== undefined) q.set("enabled", String(params.enabled))
  if (params?.status !== undefined) q.set("status", params.status)
  const qs = q.toString()
  return request<DeltaNeutralPlanListItem[]>(
    `/api/agent/delta-neutral/plans${qs ? `?${qs}` : ""}`,
  )
}

export async function getDeltaNeutralPlan(
  id: number,
): Promise<DeltaNeutralPlanListItem> {
  return request<DeltaNeutralPlanListItem>(`/api/agent/delta-neutral/plans/${id}`)
}

export interface ListDeltaNeutralSnapshotsParams {
  limit?: number
  offset?: number
}

export async function getDeltaNeutralSnapshots(
  planId: number,
  params?: ListDeltaNeutralSnapshotsParams,
): Promise<DeltaNeutralMonitorSnapshot[]> {
  const q = new URLSearchParams()
  if (params?.limit) q.set("limit", String(params.limit))
  if (params?.offset) q.set("offset", String(params.offset))
  const qs = q.toString()
  return request<DeltaNeutralMonitorSnapshot[]>(
    `/api/agent/delta-neutral/plans/${planId}/monitor-snapshots${qs ? `?${qs}` : ""}`,
  )
}

export interface ListDeltaNeutralAlertsParams {
  limit?: number
  offset?: number
}

export async function getDeltaNeutralAlerts(
  planId: number,
  params?: ListDeltaNeutralAlertsParams,
): Promise<DeltaNeutralAlert[]> {
  const q = new URLSearchParams()
  if (params?.limit) q.set("limit", String(params.limit))
  if (params?.offset) q.set("offset", String(params.offset))
  const qs = q.toString()
  return request<DeltaNeutralAlert[]>(
    `/api/agent/delta-neutral/plans/${planId}/alerts${qs ? `?${qs}` : ""}`,
  )
}

export interface ListDeltaNeutralExecutionsParams {
  limit?: number
  offset?: number
}

export async function getDeltaNeutralExecutions(
  planId: number,
  params?: ListDeltaNeutralExecutionsParams,
): Promise<DeltaNeutralExecution[]> {
  const q = new URLSearchParams()
  if (params?.limit) q.set("limit", String(params.limit))
  if (params?.offset) q.set("offset", String(params.offset))
  const qs = q.toString()
  return request<DeltaNeutralExecution[]>(
    `/api/agent/delta-neutral/plans/${planId}/executions${qs ? `?${qs}` : ""}`,
  )
}
