import { computed } from "vue";
import { useEnvironmentStore } from "@/stores/environment";
import { resolveTemplate, unresolvedTokens } from "@/lib/vars";

// 变量解析组合式：从 environment store 取出合并后的字典，
// 对所有协议客户端的 url/headers/body 等字段做 {{var}} 替换。
// 动态变量（$timestamp 等）由 resolveTemplate 内部实时计算。
export function useVarResolver() {
  const envStore = useEnvironmentStore();
  const vars = computed(() => envStore.mergedVars);

  function resolve(text: string): string {
    return resolveTemplate(text, vars.value);
  }

  function resolveHeaders(headers: Record<string, string>): Record<string, string> {
    const out: Record<string, string> = {};
    for (const [k, v] of Object.entries(headers)) out[resolve(k)] = resolve(v);
    return out;
  }

  function resolveBody(text: string): string {
    return resolve(text);
  }

  // 列出文本中引用但未定义的 token（用于发送前警告）
  function unresolved(text: string): string[] {
    return unresolvedTokens(text, vars.value);
  }

  return { vars, resolve, resolveHeaders, resolveBody, unresolved };
}
