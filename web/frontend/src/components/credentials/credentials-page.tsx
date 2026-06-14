import { IconLoader2 } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import { PageHeader } from "@/components/page-header"
import { useCredentialsPage } from "@/hooks/use-credentials-page"

import { AnthropicCredentialCard } from "./anthropic-credential-card"
import { AntigravityCredentialCard } from "./antigravity-credential-card"
import { DeviceCodeSheet } from "./device-code-sheet"
import { GeminiCredentialCard } from "./gemini-credential-card"
import { LogoutConfirmDialog } from "./logout-confirm-dialog"
import { OpenAICredentialCard } from "./openai-credential-card"

export function CredentialsPage() {
  const { t } = useTranslation()
  const {
    loading,
    error,
    activeAction,
    activeFlow,
    flowHint,
    openAIToken,
    anthropicToken,
    openaiStatus,
    anthropicStatus,
    antigravityStatus,
    geminiStatus,
    logoutDialogOpen,
    logoutConfirmProvider,
    logoutProviderLabel,
    deviceSheetOpen,
    deviceFlow,
    selectingModel,
    setOpenAIToken,
    setAnthropicToken,
    startBrowserOAuth,
    startOpenAIDeviceCode,
    stopLoading,
    saveToken,
    askLogout,
    handleConfirmLogout,
    handleLogoutDialogOpenChange,
    handleDeviceSheetOpenChange,
    selectModel,
  } = useCredentialsPage()

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.credentials")} />

      <div className="min-h-0 flex-1 overflow-y-auto px-4 sm:px-6">
        <div className="pt-2">
          <p className="text-muted-foreground text-sm">
            {t("credentials.description")}
          </p>
        </div>

        {error && (
          <div className="text-destructive bg-destructive/10 mt-4 rounded-lg px-4 py-3 text-sm">
            {error}
          </div>
        )}

        {activeFlow && (
          <div className="bg-muted mt-4 rounded-lg border px-4 py-3 text-sm">
            <p className="font-medium">{t("credentials.flow.current")}</p>
            <p className="text-muted-foreground mt-1">{flowHint}</p>
          </div>
        )}

        {loading ? (
          <div className="text-muted-foreground flex items-center gap-2 py-10 text-sm">
            <IconLoader2 className="size-4 animate-spin" />
            {t("credentials.loading")}
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4 py-5 lg:auto-rows-fr lg:grid-cols-3 xl:grid-cols-4">
            <OpenAICredentialCard
              status={openaiStatus}
              activeAction={activeAction}
              selectingModel={selectingModel}
              token={openAIToken}
              onTokenChange={setOpenAIToken}
              onStartBrowserOAuth={() => void startBrowserOAuth("openai")}
              onStartDeviceCode={() => void startOpenAIDeviceCode()}
              onStopLoading={stopLoading}
              onSaveToken={() => void saveToken("openai", openAIToken.trim())}
              onAskLogout={() => askLogout("openai")}
              onSelectModel={(modelID) => void selectModel("openai", modelID)}
            />

            <AnthropicCredentialCard
              status={anthropicStatus}
              activeAction={activeAction}
              selectingModel={selectingModel}
              token={anthropicToken}
              onTokenChange={setAnthropicToken}
              onStopLoading={stopLoading}
              onSaveToken={() =>
                void saveToken("anthropic", anthropicToken.trim())
              }
              onAskLogout={() => askLogout("anthropic")}
              onSelectModel={(modelID) =>
                void selectModel("anthropic", modelID)
              }
            />

            <AntigravityCredentialCard
              status={antigravityStatus}
              activeAction={activeAction}
              selectingModel={selectingModel}
              onStopLoading={stopLoading}
              onStartBrowserOAuth={() =>
                void startBrowserOAuth("google-antigravity")
              }
              onAskLogout={() => askLogout("google-antigravity")}
              onSelectModel={(modelID) =>
                void selectModel("google-antigravity", modelID)
              }
            />

            <GeminiCredentialCard
              status={geminiStatus}
              activeAction={activeAction}
              onStopLoading={stopLoading}
              onStartBrowserOAuth={() =>
                void startBrowserOAuth("google-gemini")
              }
              onAskLogout={() => askLogout("google-gemini")}
            />
          </div>
        )}
      </div>

      <LogoutConfirmDialog
        open={logoutDialogOpen}
        providerLabel={logoutProviderLabel}
        isSubmitting={activeAction === `${logoutConfirmProvider}:logout`}
        onOpenChange={handleLogoutDialogOpenChange}
        onConfirm={handleConfirmLogout}
      />

      <DeviceCodeSheet
        open={deviceSheetOpen}
        flow={deviceFlow}
        flowHint={flowHint}
        onOpenChange={handleDeviceSheetOpenChange}
      />
    </div>
  )
}
