<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { listAudit, type AuditLog } from "@/api/audit";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";

const props = defineProps<{ projectId?: number }>();
const { t } = useI18n();
const logs = ref<AuditLog[]>([]);
const total = ref(0);
const page = ref(1);
const perPage = 20;
const loading = ref(false);

async function load() {
  loading.value = true;
  try {
    const res = await listAudit(page.value, perPage);
    logs.value = res.logs || [];
    total.value = res.total || 0;
  } finally {
    loading.value = false;
  }
}
function fmt(ts: string) {
  const d = new Date(ts);
  return isNaN(d.getTime()) ? ts : d.toLocaleString();
}
function prev() {
  if (page.value > 1) {
    page.value--;
    load();
  }
}
function next() {
  if (page.value * perPage < total.value) {
    page.value++;
    load();
  }
}
onMounted(load);
</script>

<template>
  <div class="flex h-full flex-col gap-3 p-4">
    <div class="flex items-center justify-between">
      <h2 class="text-sm font-semibold text-foreground">{{ t("audit.title") }}</h2>
      <span class="text-xs text-muted">{{ total }} {{ t("audit.records") }}</span>
    </div>
    <Card class="flex-1 overflow-auto scroll-thin">
      <table class="w-full text-left text-xs">
        <thead class="sticky top-0 z-10 bg-surface text-muted">
          <tr>
            <th class="px-3 py-2">{{ t("audit.time") }}</th>
            <th class="px-3 py-2">{{ t("audit.user") }}</th>
            <th class="px-3 py-2">{{ t("audit.method") }}</th>
            <th class="px-3 py-2">{{ t("audit.path") }}</th>
            <th class="px-3 py-2">{{ t("audit.status") }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="l in logs" :key="l.id" class="border-t border-border/60">
            <td class="whitespace-nowrap px-3 py-2 text-muted">{{ fmt(l.createdAt) }}</td>
            <td class="px-3 py-2">{{ l.username || "#" + l.userId }}</td>
            <td class="px-3 py-2">{{ l.method }}</td>
            <td class="px-3 py-2 font-mono">{{ l.path }}</td>
            <td class="px-3 py-2">
              <span :class="l.status >= 400 ? 'text-danger' : 'text-success'">{{ l.status }}</span>
            </td>
          </tr>
          <tr v-if="!logs.length">
            <td colspan="5" class="px-3 py-6 text-center text-muted">{{ t("audit.noLogs") }}</td>
          </tr>
        </tbody>
      </table>
    </Card>
    <div class="flex items-center justify-end gap-2">
      <Button variant="ghost" :disabled="page <= 1 || loading" @click="prev">{{ t("audit.prev") }}</Button>
      <span class="text-xs text-muted">{{ page }}</span>
      <Button variant="ghost" :disabled="page * perPage >= total || loading" @click="next">{{ t("audit.next") }}</Button>
    </div>
  </div>
</template>
