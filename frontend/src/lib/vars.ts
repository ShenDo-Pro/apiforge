// 变量替换工具：支持 Postman 风格的 {{key}} 占位符。
// 仅替换「已提供值」的变量，未匹配到的 {{key}} 原样保留，方便请求里继续写模板。

import type { EnvVar } from "@/types/project";

// 每次使用都返回独立正则实例（带 g 标志的正则共享 lastIndex 有状态，
// 多调用点混用易踩坑）。resolveTemplate 走 replace、extractTokens 走 exec 循环，
// 各自用独立实例互不干扰（L13）。
function tokenRegex(): RegExp {
  return /\{\{\s*([\w.:$-]+)\s*\}\}/g;
}

// 将后端返回的 JSON 字符串解析为 EnvVar[]，解析失败回退空数组。
export function parseEnvVars(values: string | undefined | null): EnvVar[] {
  if (!values) return [];
  try {
    const arr = JSON.parse(values);
    return Array.isArray(arr) ? (arr as EnvVar[]) : [];
  } catch {
    return [];
  }
}

export function resolveTemplate(text: string, vars: Record<string, string>): string {
  if (!text) return text;
  return text.replace(tokenRegex(), (_, key: string) => {
    // 动态变量（以 $ 开头）优先于普通变量计算
    if (key.startsWith("$")) {
      const dv = dynamicValue(key);
      return dv !== undefined ? dv : `{{${key}}}`;
    }
    return Object.prototype.hasOwnProperty.call(vars, key) ? vars[key] : `{{${key}}}`;
  });
}

// 生成动态变量运行时值（Postman 风格）。返回 undefined 表示未知动态变量。
export function dynamicValue(key: string): string | undefined {
  const name = key.slice(1); // 去掉前置 $
  switch (name) {
    case "timestamp":
      return String(Math.floor(Date.now() / 1000));
    case "isoTimestamp":
      return new Date().toISOString();
    case "guid":
    case "randomUUID":
      return makeUUID();
    case "randomInt":
      return String(Math.floor(Math.random() * 1000));
    default:
      if (name.startsWith("randomInt:")) {
        const seg = name.slice("randomInt:".length);
        const dash = seg.indexOf("-");
        if (dash > 0) {
          const min = parseInt(seg.slice(0, dash), 10);
          const max = parseInt(seg.slice(dash + 1), 10);
          if (!Number.isNaN(min) && !Number.isNaN(max) && max >= min) {
            return String(min + Math.floor(Math.random() * (max - min + 1)));
          }
        }
      }
      return undefined;
  }
}

function makeUUID(): string {
  const c = (globalThis as any).crypto;
  if (c && typeof c.randomUUID === "function") return c.randomUUID();
  // 回退：基于随机数的简化 UUID
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (ch) => {
    const r = (Math.random() * 16) | 0;
    const v = ch === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

// 提取文本里出现的所有 {{key}}（去重），无论是否已定义，便于做「未解析」提示。
export function extractTokens(text: string): string[] {
  const out = new Set<string>();
  let m: RegExpExecArray | null;
  const re = tokenRegex();
  while ((m = re.exec(text))) out.add(m[1]);
  return [...out];
}

// 找出文本中引用了但未在 vars 中定义的 key。动态变量（$ 前缀）恒视为已解析。
export function unresolvedTokens(text: string, vars: Record<string, string>): string[] {
  return extractTokens(text).filter(
    (k) => !k.startsWith("$") && !Object.prototype.hasOwnProperty.call(vars, k),
  );
}
