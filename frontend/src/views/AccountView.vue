<script setup lang="ts">
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import { useToast } from "@/composables/useToast";
import { resetPassword } from "@/api/auth";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import { User as UserIcon, KeyRound } from "lucide-vue-next";

const { t } = useI18n();
const auth = useAuthStore();
const toast = useToast();

const roleLabel = computed(() =>
  auth.user?.role === "admin" ? t("account.roleAdmin") : t("account.roleUser"),
);

// 修改密码表单（复用 /auth/reset-password，C7）。
const oldPwd = ref("");
const newPwd = ref("");
const confirmPwd = ref("");
const saving = ref(false);

async function changePassword() {
  if (newPwd.value.length < 8) {
    toast.error(t("account.passwordTooShort"));
    return;
  }
  if (newPwd.value !== confirmPwd.value) {
    toast.error(t("account.passwordMismatch"));
    return;
  }
  saving.value = true;
  try {
    await resetPassword(oldPwd.value, newPwd.value);
    toast.success(t("account.changeSuccess"));
    oldPwd.value = "";
    newPwd.value = "";
    confirmPwd.value = "";
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t("account.changeFailed"));
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <div class="h-full overflow-y-auto scroll-thin">
    <div class="mx-auto max-w-3xl space-y-6 p-6">
      <h2 class="text-lg font-semibold text-foreground">{{ t("account.title") }}</h2>

      <!-- 账户信息 -->
      <div class="rounded-xl border border-border bg-surface p-4">
        <div class="mb-4 flex items-center gap-3">
          <div class="flex h-12 w-12 items-center justify-center rounded-full bg-primary/15 text-primary">
            <UserIcon :size="22" />
          </div>
          <div>
            <div class="text-base font-medium text-foreground">{{ auth.user?.username }}</div>
            <div class="text-xs text-muted">{{ roleLabel }}</div>
          </div>
        </div>
        <div class="space-y-1 text-sm">
          <div class="flex justify-between">
            <span class="text-muted">{{ t("account.username") }}</span>
            <span class="text-foreground">{{ auth.user?.username }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted">{{ t("account.role") }}</span>
            <span class="text-foreground">{{ roleLabel }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted">{{ t("account.createdAt") }}</span>
            <span class="text-foreground">{{ auth.user?.createdAt?.slice(0, 19)?.replace("T", " ") || "—" }}</span>
          </div>
        </div>
      </div>

      <!-- 修改密码 -->
      <div class="space-y-4 rounded-xl border border-border bg-surface p-4">
        <div class="flex items-center gap-2 text-sm font-medium text-muted">
          <KeyRound :size="16" /> {{ t("account.changePassword") }}
        </div>
        <p class="text-xs text-muted">{{ t("account.changePasswordHint") }}</p>
        <div class="space-y-3">
          <div>
            <label class="mb-1 block text-xs font-medium text-muted">{{ t("account.oldPassword") }}</label>
            <Input v-model="oldPwd" type="password" autocomplete="current-password" :placeholder="t('account.oldPassword')" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-muted">{{ t("account.newPassword") }}</label>
            <Input v-model="newPwd" type="password" autocomplete="new-password" :placeholder="t('account.newPassword')" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-muted">{{ t("account.confirmPassword") }}</label>
            <Input v-model="confirmPwd" type="password" autocomplete="new-password" :placeholder="t('account.confirmPassword')" />
          </div>
          <div class="flex justify-end">
            <Button :disabled="saving" @click="changePassword">
              {{ saving ? t("common.loading") : t("account.changePassword") }}
            </Button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
