import { defineStore } from "pinia";
import {
  listEnvironments,
  createEnvironment,
  updateEnvironment,
  deleteEnvironment,
  upsertGlobal,
  reorderEnvironments,
} from "@/api/environment";
import { parseEnvVars } from "@/lib/vars";
import type { EnvVar, Environment, EnvScope } from "@/types/project";

// 活动环境选择态按项目隔离，避免切换项目时串用其它项目的活动环境（M27）
function activeKeyFor(projectId: number | null): string {
  return projectId != null ? `apiforge:activeEnv:${projectId}` : "apiforge:activeEnv";
}

// 环境变量 store：服务端持久化（跟项目走、多人共享），活动环境选择态留本地。
// 多层作用域合并：Local > Environment > Collection > Global。
export interface VarSuggestion {
  key: string;
  value: string;
  scope: EnvScope;
  secret: boolean;
}

export const useEnvironmentStore = defineStore("environment", {
  state: () => ({
    environments: [] as Environment[], // 含 kind="global" 单例
    projectId: null as number | null,
    activeId: null,
    // 请求级变量（脚本/提取写回），仅内存，不落库
    localVars: {} as Record<string, string>,
    // 当前打开请求所属集合的变量（由 ProjectView 注入），已扁平化
    activeCollectionVars: {} as Record<string, string>,
    activeCollectionId: null as number | null,
    loaded: false,
  }),
  getters: {
    // 全局变量单例
    globalEnv(state): Environment | null {
      return state.environments.find((e) => e.kind === "global") ?? null;
    },
    // 当前活动环境
    activeEnv(state): Environment | null {
      return (
        state.environments.find(
          (e) => String(e.id) === state.activeId && e.kind === "env"
        ) ?? null
      );
    },
    // 全局变量字典（恒生效）
    globalVars(state): Record<string, string> {
      const out: Record<string, string> = {};
      const g = state.environments.find((e) => e.kind === "global");
      if (g) for (const v of parseEnvVars(g.values)) if (v.key.trim()) out[v.key.trim()] = v.value;
      return out;
    },
    // 集合变量字典（恒生效）
    collectionVars(state): Record<string, string> {
      return state.activeCollectionVars;
    },
    // 活动环境变量字典（仅已启用）
    environmentVars(state): Record<string, string> {
      const out: Record<string, string> = {};
      const env = state.environments.find(
        (e) => String(e.id) === state.activeId && e.kind === "env"
      );
      if (env)
        for (const v of parseEnvVars(env.values))
          if (v.enabled && v.key.trim()) out[v.key.trim()] = v.value;
      return out;
    },
    // 合并字典（发送替换用）：global → collection → environment → local
    // activeVars 为历史命名别名，保留以兼容既有调用。
    activeVars(): Record<string, string> {
      return this.mergedVars;
    },
    // 合并字典：global → collection → environment → local（同名后者覆盖前者）
    mergedVars(state): Record<string, string> {
      const merged: Record<string, string> = {};
      const g = state.environments.find((e) => e.kind === "global");
      if (g) for (const v of parseEnvVars(g.values)) if (v.key.trim()) merged[v.key.trim()] = v.value;
      for (const [k, val] of Object.entries(state.activeCollectionVars)) merged[k] = val;
      const env = state.environments.find(
        (e) => String(e.id) === state.activeId && e.kind === "env"
      );
      if (env)
        for (const v of parseEnvVars(env.values))
          if (v.enabled && v.key.trim()) merged[v.key.trim()] = v.value;
      for (const [k, val] of Object.entries(state.localVars)) merged[k] = val;
      return merged;
    },
    // 自动补全建议：所有已定义变量（含未启用），优先级决定展示值与作用域来源
    suggestions(state): VarSuggestion[] {
      const map = new Map<string, VarSuggestion>();
      const put = (scope: EnvScope, vars: EnvVar[]) => {
        for (const v of vars) {
          if (!v.key.trim()) continue;
          map.set(v.key.trim(), { key: v.key.trim(), value: v.value, scope, secret: v.secret });
        }
      };
      const g = state.environments.find((e) => e.kind === "global");
      if (g) put("global", parseEnvVars(g.values));
      put("collection", Object.entries(state.activeCollectionVars).map(([key, value]) => ({ key, value, enabled: true, secret: false })));
      const env = state.environments.find(
        (e) => String(e.id) === state.activeId && e.kind === "env"
      );
      if (env) put("environment", parseEnvVars(env.values));
      put("local", Object.entries(state.localVars).map(([key, value]) => ({ key, value, enabled: true, secret: false })));
      return [...map.values()].sort((a, b) => a.key.localeCompare(b.key));
    },
  },
  actions: {
    async fetchEnvironments(projectId: number) {
      this.projectId = projectId;
      this.environments = await listEnvironments(projectId);
      // 按当前项目读取活动环境，校验其仍存在
      const saved = localStorage.getItem(activeKeyFor(projectId));
      this.activeId =
        saved && this.environments.some((e) => String(e.id) === saved) ? saved : null;
      this.loaded = true;
    },
    setActive(id: number | null) {
      this.activeId = id === null ? null : String(id);
      const key = activeKeyFor(this.projectId);
      if (this.activeId) localStorage.setItem(key, this.activeId);
      else localStorage.removeItem(key);
    },
    // 注入当前集合的变量（递归父集合叠加），供合并与集合变量写回定位。
    setActiveCollection(collectionId: number | null, vars: Record<string, string>) {
      this.activeCollectionId = collectionId;
      this.activeCollectionVars = vars;
    },
    setLocalVar(key: string, value: string) {
      this.localVars[key] = value;
    },
    clearLocalVars() {
      this.localVars = {};
    },
    // 持久化某个环境的变量（global 走 upsertGlobal，其余走 updateEnvironment）
    async persistEnvironment(envId: number, name: string, values: EnvVar[]) {
      const env = this.environments.find((e) => e.id === envId);
      const projectId = env?.projectId;
      if (!env || projectId === undefined) return;
      if (env.kind === "global") {
        await upsertGlobal(projectId, values);
      } else {
        await updateEnvironment(projectId, envId, { name, values });
      }
      env.values = JSON.stringify(values);
      env.name = name;
    },
    async addEnvironment(name = "新环境", projectId: number) {
      const env = await createEnvironment(projectId, { name, values: [] });
      this.environments.push(env);
      this.setActive(env.id);
      return env;
    },
    async removeEnvironment(envId: number) {
      const env = this.environments.find((e) => e.id === envId);
      if (!env || env.kind === "global") return; // 全局不可删
      await deleteEnvironment(env.projectId, envId);
      this.environments = this.environments.filter((e) => e.id !== envId);
      if (this.activeId === String(envId)) this.setActive(null);
    },
    async duplicateEnvironment(envId: number) {
      const src = this.environments.find((e) => e.id === envId);
      if (!src) return null;
      const copy = await createEnvironment(src.projectId, {
        name: `${src.name} 副本`,
        values: parseEnvVars(src.values),
      });
      const idx = this.environments.findIndex((e) => e.id === envId);
      this.environments.splice(idx + 1, 0, copy);
      return copy;
    },
    async importEnvironment(payload: { name: string; values: EnvVar[] }, projectId: number) {
      const env = await createEnvironment(projectId, payload);
      this.environments.push(env);
      return env;
    },
    async reorder(ids: number[], projectId: number) {
      await reorderEnvironments(projectId, ids);
      const byId = new Map(this.environments.map((e) => [e.id, e]));
      this.environments = ids
        .map((id) => byId.get(id))
        .filter((e): e is Environment => !!e)
        .concat(this.environments.filter((e) => !ids.includes(e.id)));
    },
  },
});
