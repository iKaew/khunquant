import { IconCheck, IconLoader2 } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { OAuthModelPreset } from "@/api/oauth"
import { cn } from "@/lib/utils"

interface ModelPresetListProps {
  presets: OAuthModelPreset[]
  activeModel: string
  selectingModel: string
  onSelectModel: (modelID: string) => void
}

export function ModelPresetList({
  presets,
  activeModel,
  selectingModel,
  onSelectModel,
}: ModelPresetListProps) {
  const { t } = useTranslation()

  if (presets.length === 0) return null

  const busy = selectingModel !== ""

  return (
    <div className="space-y-1.5">
      <p className="text-muted-foreground text-xs font-medium">
        {t("credentials.model.label")}
      </p>
      <div className="flex flex-wrap gap-1.5">
        {presets.map((preset) => {
          const isActive = preset.model_id === activeModel
          const isSelecting = preset.model_id === selectingModel

          return (
            <button
              key={preset.model_id}
              type="button"
              disabled={busy}
              onClick={() => onSelectModel(preset.model_id)}
              className={cn(
                "inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors disabled:cursor-not-allowed",
                isActive
                  ? "bg-primary text-primary-foreground border-primary"
                  : "border-border text-muted-foreground hover:border-foreground/40 hover:text-foreground bg-transparent",
              )}
            >
              {isSelecting ? (
                <IconLoader2 className="size-3 animate-spin" />
              ) : isActive ? (
                <IconCheck className="size-3" />
              ) : null}
              {preset.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}
