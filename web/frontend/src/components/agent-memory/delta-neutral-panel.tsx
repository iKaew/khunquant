import { IconLoader2 } from "@tabler/icons-react"
import { useQuery } from "@tanstack/react-query"
import { useState } from "react"

import {
  type DeltaNeutralPlanListItem,
  type DeltaNeutralMonitorSnapshot,
  type DeltaNeutralAlert,
  type DeltaNeutralExecution,
  getDeltaNeutralSnapshots,
  getDeltaNeutralAlerts,
  getDeltaNeutralExecutions,
  listDeltaNeutralPlans,
} from "@/api/agent-delta-neutral"

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

function formatNum(n: number, digits = 4): string {
  return n.toLocaleString(undefined, { maximumFractionDigits: digits, minimumFractionDigits: 2 })
}

function healthLabelColor(label: string): string {
  if (!label) return "bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-400"
  const lower = label.toLowerCase()
  if (lower.includes("excellent") || lower.includes("healthy")) {
    return "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
  }
  if (lower.includes("watch") || lower.includes("warning")) {
    return "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"
  }
  if (lower.includes("danger") || lower.includes("critical")) {
    return "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
  }
  return "bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-400"
}

function healthBarColor(label: string): string {
  if (!label) return "bg-gray-400"
  const lower = label.toLowerCase()
  if (lower.includes("excellent") || lower.includes("healthy")) return "bg-green-500"
  if (lower.includes("watch") || lower.includes("warning")) return "bg-amber-500"
  if (lower.includes("danger") || lower.includes("critical")) return "bg-red-500"
  return "bg-gray-400"
}

function healthScoreTextColor(label: string): string {
  if (!label) return "text-gray-500"
  const lower = label.toLowerCase()
  if (lower.includes("excellent") || lower.includes("healthy")) return "text-green-600 dark:text-green-400"
  if (lower.includes("watch") || lower.includes("warning")) return "text-amber-600 dark:text-amber-400"
  if (lower.includes("danger") || lower.includes("critical")) return "text-red-600 dark:text-red-400"
  return "text-gray-500"
}

function SectionHeader({ title, count }: { title: string; count?: number }) {
  return (
    <div className="border-border/50 flex items-center border-b px-3 py-2">
      <span className="text-foreground/80 text-sm font-medium">{title}</span>
      {count !== undefined && (
        <span className="ml-2 rounded-full bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
          {count}
        </span>
      )}
    </div>
  )
}

function PlanSummary({ plan }: { plan: DeltaNeutralPlanListItem }) {
  const statCell = (label: string, value: string) => (
    <div className="rounded-lg border p-3">
      <div className="text-muted-foreground mb-1 text-xs">{label}</div>
      <div className="font-mono text-sm font-medium">{value}</div>
    </div>
  )

  return (
    <div className="flex flex-col gap-3">
      {/* Plan header with inline health score */}
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-foreground font-semibold">{plan.name}</span>
        <span className="text-muted-foreground text-xs">{plan.asset}</span>
        <span
          className={`rounded-full px-2 py-0.5 text-xs font-medium ${
            plan.enabled
              ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
              : "bg-muted text-muted-foreground"
          }`}
        >
          {plan.enabled ? "Active" : "Paused"}
        </span>
        {plan.cross_exchange && (
          <span className="rounded-full px-2 py-0.5 text-xs font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
            Cross-Exchange
          </span>
        )}
        {/* Inline health indicator */}
        <span className={`font-mono text-sm font-semibold ${healthScoreTextColor(plan.health_label)}`}>
          {plan.health_score}
          <span className="text-muted-foreground font-normal">/100</span>
        </span>
        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${healthLabelColor(plan.health_label)}`}>
          {plan.health_label}
        </span>
      </div>

      <div className="text-muted-foreground text-xs">
        {plan.spot_provider} ({plan.spot_account}) ↔ {plan.futures_provider} ({plan.futures_account})
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        {statCell("Capital", `${formatNum(plan.capital_usdt, 2)} USDT`)}
        {statCell("Spot Symbol", plan.spot_symbol)}
        {statCell("Futures Symbol", plan.futures_symbol)}
        {statCell("Monitor Interval", plan.monitor_interval)}
      </div>

      {/* Health progress bar */}
      <div className="h-1 w-full overflow-hidden rounded-full bg-muted">
        <div
          className={`h-full rounded-full transition-all ${healthBarColor(plan.health_label)}`}
          style={{ width: `${Math.min(100, Math.max(0, plan.health_score))}%` }}
        />
      </div>

      {/* Accumulated fees */}
      {plan.fee_snapshot && (
        <div className="rounded-lg border p-3">
          <div className="text-muted-foreground mb-2 flex items-center justify-between text-xs">
            <span>Accumulated Fees</span>
            <span>{formatDate(plan.fee_snapshot.fetched_at)}</span>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <div className="text-muted-foreground mb-0.5 text-xs">Trading Fee</div>
              <div
                className={`font-mono text-sm font-medium ${
                  plan.fee_snapshot.trading_fee_usdt < 0
                    ? "text-red-500 dark:text-red-400"
                    : "text-green-600 dark:text-green-400"
                }`}
              >
                {plan.fee_snapshot.trading_fee_usdt >= 0 ? "+" : ""}
                {formatNum(plan.fee_snapshot.trading_fee_usdt, 4)} USDT
              </div>
            </div>
            <div>
              <div className="text-muted-foreground mb-0.5 text-xs">Funding Fee</div>
              <div
                className={`font-mono text-sm font-medium ${
                  plan.fee_snapshot.funding_fee_usdt < 0
                    ? "text-red-500 dark:text-red-400"
                    : "text-green-600 dark:text-green-400"
                }`}
              >
                {plan.fee_snapshot.funding_fee_usdt >= 0 ? "+" : ""}
                {formatNum(plan.fee_snapshot.funding_fee_usdt, 4)} USDT
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function SnapshotTable({ planId }: { planId: number }) {
  const { data: snapshots, isLoading } = useQuery({
    queryKey: ["dn-snapshots", planId],
    queryFn: () => getDeltaNeutralSnapshots(planId, { limit: 50 }),
  })

  if (isLoading) {
    return (
      <div className="flex h-24 items-center justify-center">
        <IconLoader2 className="text-muted-foreground size-4 animate-spin" />
      </div>
    )
  }

  if (!snapshots || snapshots.length === 0) {
    return (
      <div className="overflow-hidden rounded-lg border">
        <SectionHeader title="Monitor Snapshots" count={0} />
        <div className="text-muted-foreground py-4 text-center text-sm">No snapshots yet.</div>
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-lg border">
      <SectionHeader title="Monitor Snapshots" count={snapshots.length} />
      <div className="max-h-56 overflow-y-auto overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="sticky top-0 z-10 bg-background">
            <tr className="bg-muted/40 text-muted-foreground border-b text-xs uppercase tracking-wide">
              <th className="px-3 py-2 text-left">Checked At</th>
              <th className="px-3 py-2 text-right">Delta Drift</th>
              <th className="px-3 py-2 text-right">Funding</th>
              <th className="px-3 py-2 text-right">Liq Dist</th>
              <th className="px-3 py-2 text-right">Margin</th>
              <th className="px-3 py-2 text-center">Health</th>
              <th className="px-3 py-2 text-center">Status</th>
            </tr>
          </thead>
          <tbody>
            {snapshots.map((snap: DeltaNeutralMonitorSnapshot) => (
              <tr key={snap.id} className="border-border/30 border-b last:border-0">
                <td className="text-muted-foreground px-3 py-2 font-mono text-xs">
                  {formatDate(snap.checked_at)}
                </td>
                <td className="px-3 py-2 text-right font-mono text-xs">
                  {formatNum(snap.delta_drift_pct, 2)}%
                </td>
                <td className="px-3 py-2 text-right font-mono text-xs">
                  {formatNum(snap.current_funding_rate, 4)}
                </td>
                <td className="px-3 py-2 text-right font-mono text-xs">
                  {formatNum(snap.liquidation_distance_pct, 2)}%
                </td>
                <td className="px-3 py-2 text-right font-mono text-xs">
                  {formatNum(snap.margin_ratio_pct, 2)}%
                </td>
                <td className="px-3 py-2 text-center">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${healthLabelColor(snap.health_label)}`}>
                    {snap.health_score}
                  </span>
                </td>
                <td className="px-3 py-2 text-center">
                  <div className="flex flex-col items-center gap-1">
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                        snap.data_status === "ok"
                          ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                          : "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
                      }`}
                    >
                      {snap.data_status}
                    </span>
                    {snap.agent_invoked && (
                      <span className="text-muted-foreground text-xs">agent invoked</span>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function AlertTable({ planId }: { planId: number }) {
  const { data: alerts, isLoading } = useQuery({
    queryKey: ["dn-alerts", planId],
    queryFn: () => getDeltaNeutralAlerts(planId, { limit: 50 }),
  })

  if (isLoading) {
    return (
      <div className="flex h-24 items-center justify-center">
        <IconLoader2 className="text-muted-foreground size-4 animate-spin" />
      </div>
    )
  }

  if (!alerts || alerts.length === 0) {
    return (
      <div className="overflow-hidden rounded-lg border">
        <SectionHeader title="Alerts" count={0} />
        <div className="text-muted-foreground py-4 text-center text-sm">No alerts yet.</div>
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-lg border">
      <SectionHeader title="Alerts" count={alerts.length} />
      <div className="max-h-48 overflow-y-auto overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="sticky top-0 z-10 bg-background">
            <tr className="bg-muted/40 text-muted-foreground border-b text-xs uppercase tracking-wide">
              <th className="px-3 py-2 text-left">Triggered At</th>
              <th className="px-3 py-2 text-center">Severity</th>
              <th className="px-3 py-2 text-left">Code</th>
              <th className="px-3 py-2 text-left">Message</th>
            </tr>
          </thead>
          <tbody>
            {alerts.map((alert: DeltaNeutralAlert) => (
              <tr key={alert.id} className="border-border/30 border-b last:border-0">
                <td className="text-muted-foreground px-3 py-2 font-mono text-xs whitespace-nowrap">
                  {formatDate(alert.triggered_at)}
                </td>
                <td className="px-3 py-2 text-center">
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                      alert.severity === "critical"
                        ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
                        : alert.severity === "warning"
                          ? "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"
                          : "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
                    }`}
                  >
                    {alert.severity}
                  </span>
                </td>
                <td className="text-muted-foreground px-3 py-2 font-mono text-xs whitespace-nowrap">
                  {alert.code}
                </td>
                <td className="px-3 py-2 text-xs">
                  <span className="block max-w-sm truncate" title={alert.message}>
                    {alert.message}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function ExecutionTable({ planId }: { planId: number }) {
  const { data: execs, isLoading } = useQuery({
    queryKey: ["dn-executions", planId],
    queryFn: () => getDeltaNeutralExecutions(planId, { limit: 50 }),
  })

  if (isLoading) {
    return (
      <div className="flex h-24 items-center justify-center">
        <IconLoader2 className="text-muted-foreground size-4 animate-spin" />
      </div>
    )
  }

  if (!execs || execs.length === 0) {
    return (
      <div className="overflow-hidden rounded-lg border">
        <SectionHeader title="Execution History" count={0} />
        <div className="text-muted-foreground py-4 text-center text-sm">No executions yet.</div>
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-lg border">
      <SectionHeader title="Execution History" count={execs.length} />
      <div className="max-h-64 overflow-y-auto overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="sticky top-0 z-10 bg-background">
            <tr className="bg-muted/40 text-muted-foreground border-b text-xs uppercase tracking-wide">
              <th className="px-3 py-2 text-left">Attempt</th>
              <th className="px-3 py-2 text-left">Requested</th>
              <th className="px-3 py-2 text-center">State</th>
              <th className="px-3 py-2 text-left">Legs</th>
            </tr>
          </thead>
          <tbody>
            {execs.map((exec: DeltaNeutralExecution) => (
              <>
                <tr key={exec.id} className="border-border/30 border-b last:border-0">
                  <td className="text-muted-foreground px-3 py-2 font-mono text-xs whitespace-nowrap">
                    {exec.attempt_id}
                  </td>
                  <td className="text-muted-foreground px-3 py-2 font-mono text-xs whitespace-nowrap">
                    {formatDate(exec.requested_at)}
                  </td>
                  <td className="px-3 py-2 text-center">
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                        exec.state === "completed"
                          ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                          : exec.state === "approved"
                            ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
                            : "bg-muted text-muted-foreground"
                      }`}
                    >
                      {exec.state}
                    </span>
                  </td>
                  <td className="px-3 py-2 text-sm">{exec.legs.length} leg(s)</td>
                </tr>
                {exec.legs.map((leg) => (
                  <tr key={`${exec.id}-leg-${leg.id}`} className="border-border/20 border-b last:border-0 bg-muted/20">
                    <td colSpan={1} className="px-6 py-1.5 text-xs">
                      {leg.leg_type}
                    </td>
                    <td colSpan={1} className="px-3 py-1.5 text-xs whitespace-nowrap">
                      {leg.side} {leg.symbol} @ {leg.provider}
                    </td>
                    <td className="px-3 py-1.5 text-center">
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          leg.state === "filled"
                            ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                            : "bg-muted text-muted-foreground"
                        }`}
                      >
                        {leg.state}
                      </span>
                    </td>
                    <td className="px-3 py-1.5 font-mono text-xs whitespace-nowrap">
                      {formatNum(leg.filled_quantity, 6)} @ {formatNum(leg.avg_fill_price, 4)}
                      <span className="text-muted-foreground ml-2">
                        ≈ {formatNum(leg.filled_quantity * leg.avg_fill_price, 2)} USDT
                      </span>
                    </td>
                  </tr>
                ))}
                {exec.error_msg && (
                  <tr key={`${exec.id}-err`} className="border-border/20 border-b last:border-0">
                    <td colSpan={4} className="px-3 pb-2 text-xs text-red-500">
                      ↳ {exec.error_msg}
                    </td>
                  </tr>
                )}
              </>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export function DeltaNeutralPanel() {
  const [selectedId, setSelectedId] = useState<number | null>(null)

  const { data: plans, isLoading } = useQuery({
    queryKey: ["dn-plans"],
    queryFn: () => listDeltaNeutralPlans(),
  })

  const selectedPlan = plans?.find((p) => p.id === selectedId)

  const itemClass = (id: number) =>
    `w-full rounded-md px-3 py-2 text-left text-sm transition-colors ${
      selectedId === id
        ? "bg-accent/80 text-foreground font-medium"
        : "text-muted-foreground hover:bg-muted/60"
    }`

  return (
    <div className="flex min-h-0 flex-1 overflow-hidden">
      {/* Left panel: plan list */}
      <div className="border-border/40 flex w-64 shrink-0 flex-col border-r">
        <div className="flex-1 overflow-auto p-2">
          {isLoading ? (
            <div className="text-muted-foreground p-2 text-sm">Loading…</div>
          ) : !plans || plans.length === 0 ? (
            <div className="text-muted-foreground p-2 text-sm">No delta-neutral plans yet.</div>
          ) : (
            <ul className="space-y-0.5">
              {plans.map((plan: DeltaNeutralPlanListItem) => (
                <li key={plan.id}>
                  <button onClick={() => setSelectedId(plan.id)} className={itemClass(plan.id)}>
                    <div className="flex items-center gap-1.5">
                      <span
                        className={`size-1.5 shrink-0 rounded-full ${plan.enabled ? "bg-green-500" : "bg-muted-foreground"}`}
                      />
                      <span className="truncate font-medium">{plan.name}</span>
                    </div>
                    <div className="text-muted-foreground mt-0.5 font-mono text-xs">
                      {plan.asset} · {plan.spot_symbol}
                    </div>
                    <div className="text-muted-foreground text-xs">
                      {plan.spot_provider} → {plan.futures_provider}
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {/* Right panel: plan detail + snapshots + alerts + executions */}
      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-4">
        {selectedId === null ? (
          <div className="text-muted-foreground flex h-full items-center justify-center text-sm">
            Select a delta-neutral plan to view details.
          </div>
        ) : selectedPlan ? (
          <>
            <PlanSummary plan={selectedPlan} />
            <SnapshotTable planId={selectedPlan.id} />
            <AlertTable planId={selectedPlan.id} />
            <ExecutionTable planId={selectedPlan.id} />
          </>
        ) : null}
      </div>
    </div>
  )
}
